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
	"gopkg.in/yaml.v3"
)

const (
	elementalComponentName = "elemental"
	elementalScriptName    = "11-elemental.sh"
	elementalConfigName    = "elemental_config.yaml"
)

//go:embed templates/11-elemental-register.sh.tpl
var elementalScript string

func configureElemental(ctx *image.Context) ([]string, error) {
	// Even if the "elemental" section is left out of the definition, the full structure will
	// be created in the definition, using defaults for the specific types. The check to determine
	// elemental registration is if the URL is present; if not, the entire elemental configuration
	// is skipped.
	if ctx.ImageDefinition.Elemental.Registration.RegistrationURL == "" {
		log.AuditComponentSkipped(elementalComponentName)
		return nil, nil
	}

	if err := writeElementalConfigFile(ctx); err != nil {
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

func writeElementalConfigFile(ctx *image.Context) error {
	configFilename := filepath.Join(ctx.CombustionDir, elementalConfigName)

	// The root of the elemental config file needs to be `elemental`, so this wrapper
	// ensures that is maintained
	type ElementalWrapper struct {
		Elemental image.Elemental
	}
	yamlData, err := yaml.Marshal(ElementalWrapper{
		Elemental: ctx.ImageDefinition.Elemental,
	})
	if err != nil {
		return fmt.Errorf("extracting elemental config: %w", err)
	}

	if err := os.WriteFile(configFilename, yamlData, fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing elemental config file %s: %w", configFilename, err)
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
