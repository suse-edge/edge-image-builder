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

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/registry"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	haulerManifestYamlName  = "hauler-manifest.yaml"
	registryScriptName      = "14-embedded-registry.sh"
	registryTarName         = "embedded-registry.tar.zst"
	registryComponentName   = "embedded artifact registry"
	registryLogFileName     = "embedded-registry.log"
	hauler                  = "hauler"
	registryDir             = "registry"
	registryPort            = "6545"
	registryMirrorsFileName = "registries.yaml"

	helmLogFileName      = "helm.log"
	helmDir              = "helm"
	helmTemplateFilename = "helm.yaml"
)

//go:embed templates/hauler-manifest.yaml.tpl
var haulerManifest string

//go:embed templates/14-embedded-registry.sh.tpl
var registryScript string

//go:embed templates/registries.yaml.tpl
var k8sRegistryMirrors string

func configureRegistry(ctx *image.Context) ([]string, error) {
	if !IsEmbeddedArtifactRegistryConfigured(ctx) {
		log.AuditComponentSkipped(registryComponentName)
		return nil, nil
	}

	registriesDir := filepath.Join(ctx.CombustionDir, registryDir)
	err := os.Mkdir(registriesDir, os.ModePerm)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("creating registry dir: %w", err)
	}

	var helmTemplatePath string
	if isComponentConfigured(ctx, filepath.Join(k8sDir, helmDir)) {
		helmTemplatePath = helmTemplateFilename

		err = generateHelmTemplate(ctx)
		if err != nil {
			log.AuditComponentFailed(registryComponentName)
			return nil, fmt.Errorf("generating helm templates and values: %w", err)
		}
	}

	var localManifestSrcDir string
	if componentDir := filepath.Join(k8sDir, "manifests"); isComponentConfigured(ctx, componentDir) {
		localManifestSrcDir = filepath.Join(ctx.ImageConfigDir, componentDir)
	}

	embeddedContainerImages := ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages
	manifestURLs := ctx.ImageDefinition.Kubernetes.Manifests.URLs
	manifestDownloadDest := ""
	if len(manifestURLs) != 0 {
		manifestDownloadDest = filepath.Join(ctx.BuildDir, "downloaded-manifests")
		err = os.Mkdir(manifestDownloadDest, os.ModePerm)
		if err != nil {
			log.AuditComponentFailed(registryComponentName)
			return nil, fmt.Errorf("creating manifest download dir: %w", err)
		}
	}

	containerImages, err := registry.GetAllImages(embeddedContainerImages, manifestURLs, localManifestSrcDir, helmTemplatePath, manifestDownloadDest)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("getting all container images: %w", err)
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		hostnames := getImageHostnames(containerImages)

		err = writeRegistryMirrors(ctx, hostnames)
		if err != nil {
			log.AuditComponentFailed(registryComponentName)
			return nil, fmt.Errorf("writing registry mirrors: %w", err)
		}
	}

	err = writeHaulerManifest(ctx, containerImages, ctx.ImageDefinition.Kubernetes.HelmCharts)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing hauler manifest: %w", err)
	}

	err = populateHaulerStore(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("populating hauler store: %w", err)
	}

	err = generateRegistryTar(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("generating hauler store tar: %w", err)
	}

	haulerBinaryPath := fmt.Sprintf("hauler-%s", string(ctx.ImageDefinition.Image.Arch))
	err = copyHaulerBinary(ctx, haulerBinaryPath)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("copying hauler binary: %w", err)
	}

	registryScriptNameResult, err := writeRegistryScript(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing registry script: %w", err)
	}

	log.AuditComponentSuccessful(registryComponentName)
	return []string{registryScriptNameResult}, nil
}

func writeHaulerManifest(ctx *image.Context, images []image.ContainerImage, charts []image.HelmChart) error {
	haulerManifestYamlFile := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	haulerDef := struct {
		ContainerImages []image.ContainerImage
		HelmCharts      []image.HelmChart
	}{
		ContainerImages: images,
		HelmCharts:      charts,
	}
	data, err := template.Parse(haulerManifestYamlName, haulerManifest, haulerDef)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", haulerManifestYamlName, err)
	}

	if err := os.WriteFile(haulerManifestYamlFile, []byte(data), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", haulerManifestYamlName, err)
	}

	return nil
}

func populateHaulerStore(ctx *image.Context) error {
	haulerManifestPath := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	args := []string{"store", "sync", "--files", haulerManifestPath}

	cmd, registryLog, err := createRegistryCommand(ctx, hauler, args)
	if err != nil {
		return fmt.Errorf("preparing to populate registry store: %w", err)
	}
	defer func() {
		if err = registryLog.Close(); err != nil {
			zap.S().Warnf("failed to close registry log file properly: %s", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("populating hauler store: %w: ", err)
	}

	return nil
}

func generateRegistryTar(ctx *image.Context) error {
	haulerTarDest := filepath.Join(ctx.CombustionDir, registryDir, registryTarName)
	args := []string{"store", "save", "--filename", haulerTarDest}

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

	return nil
}

func copyHaulerBinary(ctx *image.Context, haulerBinaryPath string) error {
	destinationDir := filepath.Join(ctx.CombustionDir, "hauler")

	err := fileio.CopyFile(haulerBinaryPath, destinationDir, fileio.ExecutablePerms)
	if err != nil {
		return fmt.Errorf("copying hauler binary to combustion dir: %w", err)
	}

	return nil
}

func writeRegistryScript(ctx *image.Context) (string, error) {
	values := struct {
		Port                string
		RegistryDir         string
		EmbeddedRegistryTar string
	}{
		Port:                registryPort,
		RegistryDir:         registryDir,
		EmbeddedRegistryTar: registryTarName,
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
	return len(ctx.ImageDefinition.Kubernetes.HelmCharts) != 0 ||
		len(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages) != 0 ||
		len(ctx.ImageDefinition.Kubernetes.Manifests.URLs) != 0
}

func getImageHostnames(containerImages []image.ContainerImage) []string {
	var hostnames []string

	for _, containerImage := range containerImages {
		result := strings.Split(containerImage.Name, "/")
		if len(result) > 1 {
			if !slices.Contains(hostnames, result[0]) && result[0] != "docker.io" {
				hostnames = append(hostnames, result[0])
			}
		}
	}

	return hostnames
}

func writeRegistryMirrors(ctx *image.Context, hostnames []string) error {
	registriesYamlFile := filepath.Join(ctx.CombustionDir, registryMirrorsFileName)
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

	if err := os.WriteFile(registriesYamlFile, []byte(data), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", registryMirrorsFileName, err)
	}

	return nil
}

func createHelmCommand(ctx *image.Context, helmCommand string) (*exec.Cmd, *os.File, error) {
	fullLogFilename := filepath.Join(ctx.BuildDir, helmLogFileName)
	logFile, err := os.OpenFile(fullLogFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening helm log file %s: %w", helmLogFileName, err)
	}

	templateFile, err := os.OpenFile(helmTemplateFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening (for append) helm template file: %w", err)
	}

	var cmd *exec.Cmd
	args := strings.Fields(helmCommand)
	if args[1] == "template" {
		cmd = exec.Command(args[0], args[1:]...)
		multiWriter := io.MultiWriter(logFile, templateFile)
		cmd.Stdout = multiWriter
	} else {
		cmd = exec.Command(args[0], args[1:]...)
		cmd.Stdout = logFile
	}

	cmd.Stderr = logFile

	return cmd, logFile, nil
}

func generateHelmTemplate(ctx *image.Context) error {
	helmSrcDir := filepath.Join(ctx.ImageConfigDir, k8sDir, helmDir)
	helmCommands, err := registry.GenerateHelmCommandsAndWriteHelmValues(helmSrcDir)
	if err != nil {
		return fmt.Errorf("generating helm templates: %w", err)
	}

	for _, command := range helmCommands {
		err := func() error {
			cmd, registryLog, err := createHelmCommand(ctx, command)
			if err != nil {
				return fmt.Errorf("preparing to template helm chart: %w", err)
			}
			defer func() {
				if err = registryLog.Close(); err != nil {
					zap.S().Warnf("failed to close helm log file properly: %s", err)
				}
			}()

			if err = cmd.Run(); err != nil {
				return fmt.Errorf("templating helm chart: %w: ", err)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}
