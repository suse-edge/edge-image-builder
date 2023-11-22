package combustion

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/context"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

const (
	customScriptsDir = "scripts"
)

func configureCustomScripts(ctx *context.Context) ([]string, error) {
	fullScriptsDir := filepath.Join(ctx.ImageConfigDir, customScriptsDir)

	// Nothing to do if the image config dir doesn't have the scripts directory
	_, err := os.Stat(fullScriptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("checking for scripts directory at %s: %w", fullScriptsDir, err)
	}

	dirListing, err := os.ReadDir(fullScriptsDir)
	if err != nil {
		return nil, fmt.Errorf("reading the scripts directory at %s: %w", fullScriptsDir, err)
	}

	// If the directory exists but there's nothing in it, consider it an error case
	if len(dirListing) == 0 {
		return nil, fmt.Errorf("no scripts found in directory %s", fullScriptsDir)
	}

	var scripts []string

	for _, scriptEntry := range dirListing {
		copyMe := filepath.Join(fullScriptsDir, scriptEntry.Name())
		copyTo := filepath.Join(ctx.CombustionDir, scriptEntry.Name())

		err = fileio.CopyFile(copyMe, copyTo, fileio.ExecutablePerms)
		if err != nil {
			return nil, fmt.Errorf("copying script to %s: %w", copyTo, err)
		}

		scripts = append(scripts, scriptEntry.Name())
	}

	return scripts, nil
}
