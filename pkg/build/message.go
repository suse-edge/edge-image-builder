package build

import (
	_ "embed"
	"fmt"
)

const (
	messageScriptName = "message.sh"
)

//go:embed scripts/message/message.sh
var messageScript string

func (b *Builder) configureMessage() error {
	err := b.writeCombustionFile(messageScriptName, messageScript, nil)
	if err != nil {
		return fmt.Errorf("copying script %s: %w", messageScriptName, err)
	}
	b.registerCombustionScript(messageScriptName)

	return nil
}
