package build

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	usersScriptName = "add-users.sh"
)

//go:embed scripts/users/add-users.sh.tpl
var usersScript string

func (b *Builder) configureUsers() error {
	// Punch out early if there are no users
	if len(b.imageConfig.OperatingSystem.Users) == 0 {
		return nil
	}

	data, err := template.Parse(usersScriptName, usersScript, b.imageConfig.OperatingSystem.Users)
	if err != nil {
		return fmt.Errorf("parsing users script template: %w", err)
	}

	filename := b.generateCombustionDirFilename(usersScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		return fmt.Errorf("writing %s to the combustion directory: %w", usersScriptName, err)
	}

	b.registerCombustionScript(usersScriptName)

	return nil
}
