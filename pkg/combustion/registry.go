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
)

const (
	haulerManifestYamlName = "hauler-manifest.yaml"
	registryScriptName     = "13-embedded-registry.sh"
	haulerTarName          = "haul.tar.zst"
	registryComponentName  = "embedded artifact registry"
)

//go:embed templates/hauler-manifest.yaml.tpl
var haulerManifest string

//go:embed templates/13-embedded-registry.sh
var registryScript string

func configureRegistry(ctx *image.Context) ([]string, error) {
	if image.IsEmbeddedArtifactRegistryEmpty(ctx.ImageDefinition.EmbeddedArtifactRegistry) {
		log.AuditComponentSkipped(registryComponentName)
		return nil, nil
	}
	haulerExecutable := filepath.Join("usr", "bin", fmt.Sprintf("hauler-%s", ctx.ImageDefinition.Image.Arch.Short()))

	err := writeHaulerManifest(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing hauler manifest: %w", err)
	}

	err = populateHaulerStore(ctx, haulerExecutable)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("populating hauler store: %w", err)
	}

	err = generateHaulerTar(ctx, haulerExecutable)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("generating hauler store tar: %w", err)
	}

	err = copyHaulerBinary(ctx, haulerExecutable)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("copying hauler binary: %w", err)
	}

	_, err = writeRegistryScript(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing registry script: %w", err)
	}

	log.AuditComponentSuccessful(registryComponentName)
	return []string{registryScriptName}, nil
}

func writeHaulerManifest(ctx *image.Context) error {
	haulerManifestYamlFile := filepath.Join(ctx.BuildDir, haulerManifestYamlName)

	data, err := template.Parse(haulerManifestYamlName, haulerManifest, ctx.ImageDefinition.EmbeddedArtifactRegistry)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", haulerManifestYamlName, err)
	}

	if err := os.WriteFile(haulerManifestYamlFile, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", haulerManifestYamlName, err)
	}

	return nil
}

func populateHaulerStore(ctx *image.Context, haulerExecutable string) error {
	haulerManifestPath := filepath.Join(ctx.BuildDir, haulerManifestYamlName)

	cmd := exec.Command(haulerExecutable, "store", "sync", "--files", haulerManifestPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("populating hauler store: %w: %s", err, string(output))
	}

	return nil
}

func generateHaulerTar(ctx *image.Context, haulerExecutable string) error {
	haulerTarDest := filepath.Join(ctx.CombustionDir, haulerTarName)

	cmd := exec.Command(haulerExecutable, "store", "save", "--filename", haulerTarDest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("creating hauler registry tar: %w: %s", err, string(output))
	}

	return nil
}

func copyHaulerBinary(ctx *image.Context, haulerExecutable string) error {
	destinationDir := filepath.Join(ctx.CombustionDir, "hauler")

	err := fileio.CopyFile(haulerExecutable, destinationDir, fileio.ExecutablePerms)
	if err != nil {
		return fmt.Errorf("copying hauler binary to combustion dir: %w", err)
	}

	return nil
}

func writeRegistryScript(ctx *image.Context) (string, error) {
	filename := filepath.Join(ctx.CombustionDir, registryScriptName)

	err := os.WriteFile(filename, []byte(registryScript), fileio.ExecutablePerms)
	if err != nil {
		return "", fmt.Errorf("writing registry script: %w", err)
	}

	return registryScriptName, nil
}
