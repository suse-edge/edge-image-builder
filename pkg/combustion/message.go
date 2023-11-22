package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	messageScriptName = "message.sh"
)

//go:embed scripts/message.sh
var messageScript string

func configureMessage(ctx *image.Context) ([]string, error) {
	filename := filepath.Join(ctx.CombustionDir, messageScriptName)

	if err := os.WriteFile(filename, []byte(messageScript), fileio.ExecutablePerms); err != nil {
		return nil, fmt.Errorf("copying script %s: %w", messageScriptName, err)
	}

	return []string{messageScriptName}, nil
}
