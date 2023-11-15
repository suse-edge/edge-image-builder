package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

const (
	scriptsDir = "scripts"
	scriptMode = 0o744
)

func (b *Builder) configureScripts() error {
	fullScriptsDir := filepath.Join(b.dirStructure.ImageConfigDir, scriptsDir)

	// Nothing to do if the image config dir doesn't have the scripts directory
	_, err := os.Stat(fullScriptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking for scripts directory at %s: %w", fullScriptsDir, err)
	}

	dirListing, err := os.ReadDir(fullScriptsDir)
	if err != nil {
		return fmt.Errorf("reading the scripts directory at %s: %w", fullScriptsDir, err)
	}

	// If the directory exists but there's nothing in it, consider it an error case
	if len(dirListing) == 0 {
		return fmt.Errorf("no scripts found in directory %s", fullScriptsDir)
	}

	for _, scriptEntry := range dirListing {
		copyMe := filepath.Join(fullScriptsDir, scriptEntry.Name())
		copyTo := filepath.Join(b.dirStructure.CombustionDir, scriptEntry.Name())

		err = fileio.CopyFile(copyMe, copyTo)
		if err != nil {
			return fmt.Errorf("copying script to %s: %w", copyTo, err)
		}
		err = os.Chmod(copyTo, scriptMode)
		if err != nil {
			return fmt.Errorf("modifying permissions for script %s: %w", copyTo, err)
		}

		// Make sure the combustion main script will execute the newly copied script
		b.registerCombustionScript(scriptEntry.Name())
	}

	return nil
}
