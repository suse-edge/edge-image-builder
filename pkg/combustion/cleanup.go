package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"go.uber.org/zap"
)

const (
	cleanupScriptName    = "cleanup-combustion.sh"
	cleanupComponentName = "cleanup"
)

//go:embed templates/cleanup-combustion.sh
var cleanupScript string

func configureCleanup(ctx *image.Context) ([]string, error) {
	if ctx.ImageDefinition.Image.ImageType != image.TypeRAW {
		log.AuditComponentSkipped(cleanupComponentName)
		zap.S().Info("skipping cleanup component, image type is not raw")
		return nil, nil
	}

	cleanupScriptFilename := filepath.Join(ctx.CombustionDir, cleanupScriptName)
	if err := os.WriteFile(cleanupScriptFilename, []byte(cleanupScript), fileio.ExecutablePerms); err != nil {
		return nil, fmt.Errorf("writing cleanup files script %s: %w", cleanupScriptName, err)
	}

	log.AuditComponentSuccessful(cleanupComponentName)
	return []string{cleanupScriptName}, nil
}
