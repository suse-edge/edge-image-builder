package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	registryScriptName      = "26-embedded-registry.sh"
	registryTarSuffix       = "registry.tar.zst"
	registryComponentName   = "embedded artifact registry"
	registryLogFileName     = "embedded-registry.log"
	hauler                  = "hauler"
	registryDir             = "registry"
	registryPort            = "6545"
	registryMirrorsFileName = "registries.yaml"

	HelmDir   = "helm"
	ValuesDir = "values"
	CertsDir  = "certs"
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

	configured, err := c.configureEmbeddedArtifactRegistry(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("configuring embedded artifact registry: %w", err)
	}

	if !configured {
		log.AuditComponentSkipped(registryComponentName)
		zap.S().Info("Skipping embedded artifact registry since the provided manifests/helm charts contain no images")
		return nil, nil
	}

	script, err := writeRegistryScript(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing registry script: %w", err)
	}

	log.AuditComponentSuccessful(registryComponentName)
	return []string{script}, nil
}

func addImageToHauler(ctx *image.Context, containerImage string) error {
	args := []string{"store", "add", "image", containerImage, "-p", fmt.Sprintf("linux/%s", ctx.ImageDefinition.Image.Arch.Short())}

	cmd, registryLog, err := createRegistryCommand(ctx, hauler, args)
	if err != nil {
		return fmt.Errorf("preparing to add image to hauler store: %w", err)
	}
	defer func() {
		if err = registryLog.Close(); err != nil {
			zap.S().Warnf("failed to close registry log file properly: %s", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("running hauler add image command: %w: ", err)
	}

	return nil
}

func generateRegistryTar(ctx *image.Context, imageTarDest string) error {
	args := []string{"store", "save", "--filename", imageTarDest}

	cmd, registryLog, err := createRegistryCommand(ctx, hauler, args)
	if err != nil {
		return fmt.Errorf("preparing to generate registry tar: %w", err)
	}
	defer func() {
		if err = registryLog.Close(); err != nil {
			zap.S().Warnf("failed to close registry log file properly: %s", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("creating registry tar: %w: ", err)
	}

	if err = os.RemoveAll("store"); err != nil {
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

func createRegistryCommand(ctx *image.Context, commandName string, args []string) (*exec.Cmd, *os.File, error) {
	fullLogFilename := filepath.Join(ctx.BuildDir, registryLogFileName)
	logFile, err := os.OpenFile(fullLogFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening registry log file %s: %w", registryLogFileName, err)
	}

	cmd := exec.Command(commandName, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd, logFile, nil
}

func IsEmbeddedArtifactRegistryConfigured(ctx *image.Context) bool {
	return len(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages) != 0 ||
		len(ctx.ImageDefinition.Kubernetes.Manifests.URLs) != 0 ||
		len(ctx.ImageDefinition.Kubernetes.Helm.Charts) != 0 ||
		isComponentConfigured(ctx, filepath.Join(K8sDir, K8sManifestsDir))
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

func (c *Combustion) configureEmbeddedArtifactRegistry(ctx *image.Context) (bool, error) {
	images, err := c.Registry.ContainerImages()
	if err != nil {
		return false, fmt.Errorf("extracting container images: %w", err)
	}

	if len(images) == 0 {
		return false, nil
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		hostnames := getImageHostnames(images)

		if err = writeRegistryMirrors(ctx, hostnames); err != nil {
			return false, fmt.Errorf("writing registry mirrors: %w", err)
		}
	}

	artefactsPath := registryArtefactsPath(ctx)
	if err = os.Mkdir(artefactsPath, os.ModePerm); err != nil {
		return false, fmt.Errorf("creating registry dir: %w", err)
	}

	if err = populateRegistry(ctx, images); err != nil {
		return false, fmt.Errorf("populating registry: %w", err)
	}

	sourcePath := "/usr/bin/hauler"
	destinationPath := filepath.Join(registryArtefactsPath(ctx), "hauler")
	if err = fileio.CopyFile(sourcePath, destinationPath, fileio.ExecutablePerms); err != nil {
		return false, fmt.Errorf("copying hauler binary: %w", err)
	}

	return true, nil
}

func registryArtefactsPath(ctx *image.Context) string {
	return filepath.Join(ctx.ArtefactsDir, registryDir)
}

func populateRegistry(ctx *image.Context, images []string) error {
	bar := progressbar.Default(int64(len(images)), "Populating Embedded Artifact Registry...")
	zap.S().Infof("Adding the following images to the embedded artifact registry:\n%s", images)

	for _, i := range images {
		if err := addImageToHauler(ctx, i); err != nil {
			return fmt.Errorf("adding image to hauler: %w", err)
		}

		convertedImage := strings.ReplaceAll(i, "/", "_")
		convertedImageName := fmt.Sprintf("%s-%s", convertedImage, registryTarSuffix)

		imageTarDest := filepath.Join(registryArtefactsPath(ctx), convertedImageName)
		if err := generateRegistryTar(ctx, imageTarDest); err != nil {
			return fmt.Errorf("generating hauler store tar: %w", err)
		}

		if err := bar.Add(1); err != nil {
			zap.S().Debugf("Error incrementing the progress bar: %s", err)
		}
	}

	return nil
}
