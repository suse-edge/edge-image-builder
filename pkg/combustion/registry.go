package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	registryScriptName    = "13-embedded-registry.sh"
	registryComponentName = "embedded artifact registry"
	registryLogFileName   = "embedded-registry.log"
	hauler                = "hauler"
)

//go:embed templates/hauler-manifest.yaml.tpl
var haulerManifest string

//go:embed templates/13-embedded-registry.sh.tpl
var registryScript string

func configureRegistry(ctx *image.Context) ([]string, error) {
	if IsEmbeddedArtifactRegistryAndKubernetesManifestsEmpty(ctx) {
		log.AuditComponentSkipped(registryComponentName)
		return nil, nil
	}

	haulerBinaryPath := fmt.Sprintf("hauler-%s", string(ctx.ImageDefinition.Image.Arch))
	err := copyHaulerBinary(ctx, haulerBinaryPath)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("copying hauler binary: %w", err)
	}

	registryScriptNameResult, err := writeRegistryScript(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing registry script: %w", err)
	}

	registriesDir := filepath.Join(ctx.CombustionDir, "registry")
	err = os.Mkdir(registriesDir, os.ModePerm)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("creating registry dir: %w", err)
	}

	if len(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages) != 0 {
		err = writeHaulerManifestAndGenerateTar(ctx, ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages, nil, "embedded-artifact-registry")
		if err != nil {
			log.AuditComponentFailed(registryComponentName)
			return nil, fmt.Errorf("writing hauler manifest and generating registry tar: %w", err)
		}
	}

	log.AuditComponentSuccessful(registryComponentName)
	return []string{registryScriptNameResult}, nil
}

func writeHaulerManifest(ctx *image.Context, images []image.ContainerImage, charts []image.HelmChart, haulerManifestYamlName string) error {
	haulerManifestYamlFile := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	registryDef := image.EmbeddedArtifactRegistry{ContainerImages: images, HelmCharts: charts}

	data, err := template.Parse(haulerManifestYamlName, haulerManifest, registryDef)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", haulerManifestYamlName, err)
	}

	if err := os.WriteFile(haulerManifestYamlFile, []byte(data), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", haulerManifestYamlName, err)
	}

	return nil
}

func populateHaulerStore(ctx *image.Context, haulerManifestYamlName string) error {
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

func generateRegistryTar(ctx *image.Context, registryTarName string) error {
	haulerTarDest := filepath.Join(ctx.CombustionDir, "registry", registryTarName)
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
		Port string
	}{
		Port: "6545",
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

func IsEmbeddedArtifactRegistryAndKubernetesManifestsEmpty(ctx *image.Context) bool {
	return len(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages) == 0 && len(ctx.ImageDefinition.Kubernetes.HelmCharts) == 0
}

func writeHaulerManifestAndGenerateTar(ctx *image.Context, images []image.ContainerImage, charts []image.HelmChart, registryOrigin string) error {
	haulerManifestYamlName := fmt.Sprintf("%s.yaml", registryOrigin)
	registryTarName := fmt.Sprintf("%s.tar.zst", registryOrigin)

	err := writeHaulerManifest(ctx, images, charts, haulerManifestYamlName)
	if err != nil {
		return fmt.Errorf("writing hauler manifest: %w", err)
	}

	err = populateHaulerStore(ctx, haulerManifestYamlName)
	if err != nil {
		return fmt.Errorf("populating hauler store: %w", err)
	}

	err = generateRegistryTar(ctx, registryTarName)
	if err != nil {
		return fmt.Errorf("generating hauler store tar: %w", err)
	}

	return nil
}
