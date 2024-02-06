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
)

const (
	keymapComponentName = "keymap"
	keymapScriptName    = "12-keymap-setup.sh"
)

//go:embed templates/12-keymap-setup.sh.tpl
var keymapScript string

func configureKeymap(ctx *image.Context) ([]string, error) {
	if err := writeKeymapCombustionScript(ctx); err != nil {
		log.AuditComponentFailed(keymapComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(keymapComponentName)
	return []string{keymapScriptName}, nil
}

func writeKeymapCombustionScript(ctx *image.Context) error {
	keymapScriptFilename := filepath.Join(ctx.CombustionDir, keymapScriptName)

	data, err := template.Parse(keymapScriptName, keymapScript, ctx.ImageDefinition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", keymapScriptName, err)
	}

	if err := os.WriteFile(keymapScriptFilename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", keymapScriptFilename, err)
	}
	return nil
}
