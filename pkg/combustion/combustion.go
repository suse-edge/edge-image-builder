package combustion

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

// configureComponent defines the combustion component contract.
// Each component (e.g. "users") receives the necessary dir structure and
// additional values it should be operating with through a Context object.
//
// configureComponent returns a slice of scripts which should be executed as part of the Combustion script.
// Result can also be an empty slice or nil if this is not necessary.
type configureComponent func(context *image.Context) ([]string, error)

// Configure iterates over all separate Combustion components and configures them independently.
// If all of those are successful, the Combustion script is assembled and written to the file system.
func Configure(ctx *image.Context) error {
	var combustionScripts []string

	combustionComponents := map[string]configureComponent{
		"message": configureMessage,
		"users":   configureUsers,
		"rpm":     configureRPMs,
		"custom":  configureCustomScripts,
	}

	for componentName, configureFunc := range combustionComponents {
		scripts, err := configureFunc(ctx)
		if err != nil {
			return fmt.Errorf("configuring component %q: %w", componentName, err)
		}

		combustionScripts = append(combustionScripts, scripts...)
	}

	script, err := assembleScript(combustionScripts)
	if err != nil {
		return fmt.Errorf("assembling script: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, "script")
	if err = os.WriteFile(filename, []byte(script), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing script: %w", err)
	}

	return nil
}
