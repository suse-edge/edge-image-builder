package build

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

const (
	messageScriptName = "message.sh"
)

//go:embed scripts/message/message.sh
var messageScript string

func (b *Builder) configureMessage() error {
	filename := b.generateCombustionDirFilename(messageScriptName)

	if err := os.WriteFile(filename, []byte(messageScript), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("copying script %s: %w", messageScriptName, err)
	}
	b.registerCombustionScript(messageScriptName)

	return nil
}
