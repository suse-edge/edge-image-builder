package build

import (
	_ "embed"
	"fmt"
	"os"
)

const (
	usersScriptName = "add-users.sh"
	userScriptMode  = 0o744
)

//go:embed scripts/users/add-users.sh.tpl
var usersScript string

func (b *Builder) configureUsers() error {
	// Punch out early if there are no users
	if len(b.imageConfig.OperatingSystem.Users) == 0 {
		return nil
	}

	filename, err := b.writeCombustionFile(usersScriptName, usersScript, b.imageConfig.OperatingSystem.Users)
	if err != nil {
		return fmt.Errorf("writing %s to the combustion directory: %w", usersScriptName, err)
	}
	err = os.Chmod(filename, userScriptMode)
	if err != nil {
		return fmt.Errorf("modifying permissions for script %s: %w", filename, err)
	}

	b.registerCombustionScript(usersScriptName)

	return nil
}
