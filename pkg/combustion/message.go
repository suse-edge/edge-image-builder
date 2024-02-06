package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

const (
	messageScriptName    = "48-message.sh"
	messageComponentName = "identifier"
)

//go:embed templates/48-message.sh
var messageScript string

func configureMessage(ctx *image.Context) ([]string, error) {
	filename := filepath.Join(ctx.CombustionDir, messageScriptName)

	if err := os.WriteFile(filename, []byte(messageScript), fileio.ExecutablePerms); err != nil {
		log.AuditComponentFailed(messageComponentName)
		return nil, fmt.Errorf("copying script %s: %w", messageScriptName, err)
	}

	log.AuditComponentSuccessful(messageComponentName)
	return []string{messageScriptName}, nil
}
