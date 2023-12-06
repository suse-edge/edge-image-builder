package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	elementalComponentName = "elemental"
	elementalConfigDir     = "elemental"
	elementalScriptName    = "11-elemental.sh"
	elementalConfigName    = "elemental_config.yaml"
)

//go:embed templates/11-elemental-register.sh.tpl
var elementalScript string

func configureElemental(ctx *image.Context) ([]string, error) {

	if !isComponentConfigured(ctx, elementalConfigDir) {
		log.AuditComponentSkipped(elementalComponentName)
		zap.L().Info("Skipping elemental registration component. Configuration is not provided")
		return nil, nil
	}

	if err := copyElementalConfigFile(ctx); err != nil {
		log.AuditComponentFailed(elementalComponentName)
		return nil, err
	}

	if err := writeElementalCombustionScript(ctx); err != nil {
		log.AuditComponentFailed(elementalComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(elementalComponentName)
	return []string{elementalScriptName}, nil
}

func copyElementalConfigFile(ctx *image.Context) error {
	srcFile := filepath.Join(ctx.ImageConfigDir, elementalConfigDir, elementalConfigName)
	destFile := filepath.Join(ctx.CombustionDir, elementalConfigName)

	err := fileio.CopyFile(srcFile, destFile, fileio.NonExecutablePerms)
	if err != nil {
		return fmt.Errorf("error copying elemental config file %s: %w", srcFile, err)
	}

	return nil
}

func writeElementalCombustionScript(ctx *image.Context) error {
	elementalScriptFilename := filepath.Join(ctx.CombustionDir, elementalScriptName)

	values := struct {
		ConfigFile string
	}{
		ConfigFile: elementalConfigName,
	}
	data, err := template.Parse(elementalScriptName, elementalScript, &values)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", elementalScriptName, err)
	}

	if err := os.WriteFile(elementalScriptFilename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", elementalScriptFilename, err)
	}
	return nil
}
