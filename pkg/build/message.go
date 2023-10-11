package build

import (
	"fmt"
)

const (
	messageScriptName = "message.sh"
	messageScriptsDir = "message"
)

func (b *Builder) configureMessage() error {
	err := b.copyCombustionFile(messageScriptsDir, messageScriptName)
	if err != nil {
		return fmt.Errorf("copying script %s: %w", messageScriptName, err)
	}

	return nil
}
