package combustion

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	registryScriptName      = "26-embedded-registry.sh"
	registryTarSuffix       = "registry.tar.zst"
	registryComponentName   = "embedded artifact registry"
	hauler                  = "hauler"
	registryDir             = "registry"
	registryPort            = "6545"
	registryMirrorsFileName = "registries.yaml"
)

var (
	//go:embed templates/26-embedded-registry.sh.tpl
	registryScript string

	//go:embed templates/registries.yaml.tpl
	k8sRegistryMirrors string
)

func (c *Combustion) configureRegistry(ctx *image.Context) ([]string, error) {
	if !IsEmbeddedArtifactRegistryConfigured(ctx) {
		log.AuditComponentSkipped(registryComponentName)
		return nil, nil
	}

	images, err := c.Registry.ContainerImages()
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("extracting container images: %w", err)
	}

	if len(images) == 0 {
		log.AuditComponentSkipped(registryComponentName)
		zap.S().Info("Skipping embedded artifact registry since the provided manifests/helm charts contain no images")
		return nil, nil
	}

	script, err := c.configureEmbeddedArtifactRegistry(ctx, images)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("configuring embedded artifact registry: %w", err)
	}

	log.AuditComponentSuccessful(registryComponentName)
	return []string{script}, nil
}

func storeImage(containerImage, arch string, outputWriter io.Writer) error {
	args := []string{"store", "add", "image", containerImage, "-p", fmt.Sprintf("linux/%s", arch)}

	cmd := exec.Command(hauler, args...)
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter

	return cmd.Run()
}

func generateRegistryTar(imageTarDest string, outputWriter io.Writer) error {
	args := []string{"store", "save", "--filename", imageTarDest}

	cmd := exec.Command(hauler, args...)
	cmd.Stdout = outputWriter
	cmd.Stderr = outputWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("creating registry tarball: %w: ", err)
	}

	if err := os.RemoveAll("store"); err != nil {
		return fmt.Errorf("removing registry store: %w", err)
	}

	return nil
}

func writeRegistryScript(ctx *image.Context) (string, error) {
	values := struct {
		RegistryPort      string
		RegistryDir       string
		RegistryTarSuffix string
	}{
		RegistryPort:      registryPort,
		RegistryDir:       prependArtefactPath(registryDir),
		RegistryTarSuffix: registryTarSuffix,
	}

	data, err := template.Parse(registryScriptName, registryScript, &values)
	if err != nil {
		return "", fmt.Errorf("parsing registry script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, registryScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		return "", fmt.Errorf("writing registry script: %w", err)
	}

	return registryScriptName, nil
}

func IsEmbeddedArtifactRegistryConfigured(ctx *image.Context) bool {
	return len(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages) != 0 ||
		len(ctx.ImageDefinition.Kubernetes.Manifests.URLs) != 0 ||
		len(ctx.ImageDefinition.Kubernetes.Helm.Charts) != 0 ||
		isComponentConfigured(ctx, localKubernetesManifestsPath())
}

func getImageHostnames(containerImages []string) []string {
	var hostnames []string

	for _, containerImage := range containerImages {
		result := strings.Split(containerImage, "/")
		if len(result) > 1 {
			if !slices.Contains(hostnames, result[0]) && result[0] != "docker.io" {
				hostnames = append(hostnames, result[0])
			}
		}
	}

	return hostnames
}

func writeRegistryMirrors(ctx *image.Context, hostnames []string) error {
	artefactsPath := kubernetesArtefactsPath(ctx)
	if err := os.MkdirAll(artefactsPath, os.ModePerm); err != nil {
		return fmt.Errorf("creating kubernetes artefacts path: %w", err)
	}

	registriesYamlFile := filepath.Join(artefactsPath, registryMirrorsFileName)
	registriesDef := struct {
		Hostnames []string
		Port      string
	}{
		Hostnames: hostnames,
		Port:      registryPort,
	}

	data, err := template.Parse(registryMirrorsFileName, k8sRegistryMirrors, registriesDef)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", registryMirrorsFileName, err)
	}

	if err = os.WriteFile(registriesYamlFile, []byte(data), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", registryMirrorsFileName, err)
	}

	return nil
}

func (c *Combustion) configureEmbeddedArtifactRegistry(ctx *image.Context, containerImages []string) (string, error) {
	if len(containerImages) == 0 {
		return "", fmt.Errorf("no container images specified")
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		hostnames := getImageHostnames(containerImages)

		if err := writeRegistryMirrors(ctx, hostnames); err != nil {
			return "", fmt.Errorf("writing registry mirrors: %w", err)
		}
	}

	artefactsPath := registryArtefactsPath(ctx)
	if err := os.Mkdir(artefactsPath, os.ModePerm); err != nil {
		return "", fmt.Errorf("creating registry dir: %w", err)
	}

	if err := populateRegistry(ctx, containerImages); err != nil {
		return "", fmt.Errorf("populating registry: %w", err)
	}

	sourcePath := "/usr/bin/hauler"
	destinationPath := filepath.Join(registryArtefactsPath(ctx), "hauler")
	if err := fileio.CopyFile(sourcePath, destinationPath, fileio.ExecutablePerms); err != nil {
		return "", fmt.Errorf("copying hauler binary: %w", err)
	}

	script, err := writeRegistryScript(ctx)
	if err != nil {
		return "", fmt.Errorf("writing registry script: %w", err)
	}

	return script, nil
}

func registryArtefactsPath(ctx *image.Context) string {
	return filepath.Join(ctx.ArtefactsDir, registryDir)
}

func populateRegistry(ctx *image.Context, images []string) error {
	p, err := podman.New(ctx.BuildDir)
	if err != nil {
		zap.S().Warnf("Setting up Podman instance: %v", err)
	}

	imageCacheDir := filepath.Join(ctx.CacheDir, "images")
	if err = os.MkdirAll(imageCacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating container image cache dir: %w", err)
	}

	bar := progressbar.Default(int64(len(images)), "Populating Embedded Artifact Registry...")
	zap.S().Infof("Adding the following images to the embedded artifact registry:\n%s", images)

	const registryLogFileName = "embedded-registry.log"
	logFilename := filepath.Join(ctx.BuildDir, registryLogFileName)

	logFile, err := os.OpenFile(logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return fmt.Errorf("opening registry log file: %w", err)
	}

	defer func() {
		if err = logFile.Close(); err != nil {
			zap.S().Warnf("Failed to close registry log file properly: %v", err)
		}
	}()

	arch := ctx.ImageDefinition.Image.Arch.Short()

	for _, img := range images {
		convertedImage := strings.ReplaceAll(img, "/", "_")
		convertedImageName := fmt.Sprintf("%s-%s", convertedImage, registryTarSuffix)
		if strings.Contains(img, ":latest") {
			var digest string
			digest, err = p.Inspect(img, arch)
			if err != nil {
				zap.S().Warnf("Failed getting digest for %s: %s", img, err)

				// In the case where we're not able to find a digest, we'll use a timestamp to prevent staleness
				digest = fmt.Sprintf("%d", time.Now().Unix())
			}

			convertedImageName = fmt.Sprintf("%s-%s-%s", convertedImage, digest, registryTarSuffix)
		}

		imageCacheLocation := filepath.Join(imageCacheDir, convertedImageName)
		imageTarDest := filepath.Join(registryArtefactsPath(ctx), convertedImageName)

		if fileio.FileExists(imageCacheLocation) {
			if err = fileio.CopyFile(imageCacheLocation, imageTarDest, fileio.NonExecutablePerms); err != nil {
				return fmt.Errorf("copying cached container image: %w", err)
			}
		} else {
			if err = storeImage(img, arch, logFile); err != nil {
				return fmt.Errorf("adding image to registry store: %w", err)
			}

			if err = generateRegistryTar(imageTarDest, logFile); err != nil {
				return fmt.Errorf("generating registry store tarball: %w", err)
			}

			if err = fileio.CopyFile(imageTarDest, imageCacheLocation, fileio.NonExecutablePerms); err != nil {
				return fmt.Errorf("copying container image to cache: %w", err)
			}
		}
		if err = bar.Add(1); err != nil {
			zap.S().Debugf("Error incrementing the progress bar: %s", err)
		}
	}

	return nil
}
