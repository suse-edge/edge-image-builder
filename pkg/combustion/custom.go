package combustion

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

const (
	customDir           = "custom"
	customScriptsDir    = "scripts"
	customFilesDir      = "files"
	customComponentName = "custom files"
)

func configureCustomFiles(ctx *image.Context) ([]string, error) {
	if !isComponentConfigured(ctx, customDir) {
		log.AuditComponentSkipped(customComponentName)
		return nil, nil
	}

	err := handleCustomFiles(ctx)
	if err != nil {
		log.AuditComponentFailed(customComponentName)
		return nil, err
	}

	scripts, err := handleCustomScripts(ctx)
	if err != nil {
		log.AuditComponentFailed(customComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(customComponentName)
	return scripts, nil
}

func handleCustomFiles(ctx *image.Context) error {
	fullFilesDir := generateComponentPath(ctx, filepath.Join(customDir, customFilesDir))
	_, err := copyCustomFiles(fullFilesDir, ctx.CombustionDir, fileio.NonExecutablePerms)
	return err
}

func handleCustomScripts(ctx *image.Context) ([]string, error) {
	fullScriptsDir := generateComponentPath(ctx, filepath.Join(customDir, customScriptsDir))
	scripts, err := copyCustomFiles(fullScriptsDir, ctx.CombustionDir, fileio.ExecutablePerms)
	return scripts, err
}

func copyCustomFiles(fromDir, combustionDir string, params os.FileMode) ([]string, error) {
	if _, err := os.Stat(fromDir); os.IsNotExist(err) {
		return nil, nil
	}

	dirListing, err := os.ReadDir(fromDir)
	if err != nil {
		return nil, fmt.Errorf("reading the custom directory at %s: %w", fromDir, err)
	}

	// If the directory exists but there's nothing in it, consider it an error case
	if len(dirListing) == 0 {
		return nil, fmt.Errorf("no files found in directory %s", fromDir)
	}

	var copiedFiles []string

	for _, scriptEntry := range dirListing {
		copyMe := filepath.Join(fromDir, scriptEntry.Name())
		copyTo := filepath.Join(combustionDir, scriptEntry.Name())

		err = fileio.CopyFile(copyMe, copyTo, params)
		if err != nil {
			return nil, fmt.Errorf("copying script to %s: %w", copyTo, err)
		}

		copiedFiles = append(copiedFiles, scriptEntry.Name())
	}

	return copiedFiles, nil

}
