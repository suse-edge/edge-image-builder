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
	err := copyCustomFiles(fullFilesDir, ctx.CombustionDir)
	return err
}

func handleCustomScripts(ctx *image.Context) ([]string, error) {
	fullScriptsDir := generateComponentPath(ctx, filepath.Join(customDir, customScriptsDir))
	executablePerms := fileio.ExecutablePerms
	scripts, err := copyCustomScripts(fullScriptsDir, ctx.CombustionDir, &executablePerms)
	return scripts, err
}

func copyCustomFiles(fromDir, toDir string) error {
	if _, err := os.Stat(fromDir); os.IsNotExist(err) {
		return nil
	}

	dirEntries, err := os.ReadDir(fromDir)
	if err != nil {
		return fmt.Errorf("reading the custom files directory at %s: %w", fromDir, err)
	}

	// If the directory exists but there's nothing in it, consider it an error case
	if len(dirEntries) == 0 {
		return fmt.Errorf("no files found in directory %s", fromDir)
	}

	if err = fileio.CopyFiles(fromDir, toDir, "", true); err != nil {
		return fmt.Errorf("copying custom: %w", err)
	}

	return nil
}

func copyCustomScripts(fromDir, toDir string, filePermissions *os.FileMode) ([]string, error) {
	if _, err := os.Stat(fromDir); os.IsNotExist(err) {
		return nil, nil
	}

	dirEntries, err := os.ReadDir(fromDir)
	if err != nil {
		return nil, fmt.Errorf("reading the custom scripts directory at %s: %w", fromDir, err)
	}

	// If the directory exists but there's nothing in it, consider it an error case
	if len(dirEntries) == 0 {
		return nil, fmt.Errorf("no scripts found in directory %s", fromDir)
	}

	var copiedFiles []string

	for _, entry := range dirEntries {
		copyMe := filepath.Join(fromDir, entry.Name())
		copyTo := filepath.Join(toDir, entry.Name())

		if err = fileio.CopyFile(copyMe, copyTo, *filePermissions); err != nil {
			return nil, fmt.Errorf("copying script to %s: %w", copyTo, err)
		}

		copiedFiles = append(copiedFiles, entry.Name())
	}

	return copiedFiles, nil

}
