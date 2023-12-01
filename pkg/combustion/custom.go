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
	customScriptsDir    = "scripts"
	customComponentName = "custom scripts"
)

func configureCustomScripts(ctx *image.Context) ([]string, error) {
	if !isComponentConfigured(ctx, customScriptsDir) {
		log.AuditComponentSkipped(customComponentName)
		return nil, nil
	}

	fullScriptsDir := generateComponentPath(ctx, customScriptsDir)

	dirListing, err := os.ReadDir(fullScriptsDir)
	if err != nil {
		log.AuditComponentFailed(customComponentName)
		return nil, fmt.Errorf("reading the scripts directory at %s: %w", fullScriptsDir, err)
	}

	// If the directory exists but there's nothing in it, consider it an error case
	if len(dirListing) == 0 {
		log.AuditComponentFailed(customComponentName)
		return nil, fmt.Errorf("no scripts found in directory %s", fullScriptsDir)
	}

	var scripts []string

	for _, scriptEntry := range dirListing {
		copyMe := filepath.Join(fullScriptsDir, scriptEntry.Name())
		copyTo := filepath.Join(ctx.CombustionDir, scriptEntry.Name())

		err = fileio.CopyFile(copyMe, copyTo, fileio.ExecutablePerms)
		if err != nil {
			log.AuditComponentFailed(customComponentName)
			return nil, fmt.Errorf("copying script to %s: %w", copyTo, err)
		}

		scripts = append(scripts, scriptEntry.Name())
	}

	log.AuditComponentSuccessful(customComponentName)
	return scripts, nil
}
