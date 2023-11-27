package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	usersScriptName = "add-users.sh"
)

//go:embed scripts/add-users.sh.tpl
var usersScript string

func configureUsers(ctx *image.Context) ([]string, error) {
	// Punch out early if there are no users
	if len(ctx.ImageDefinition.OperatingSystem.Users) == 0 {
		return nil, nil
	}

	data, err := template.Parse(usersScriptName, usersScript, ctx.ImageDefinition.OperatingSystem.Users)
	if err != nil {
		return nil, fmt.Errorf("parsing users script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, usersScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		return nil, fmt.Errorf("writing %s to the combustion directory: %w", usersScriptName, err)
	}

	return []string{usersScriptName}, nil
}
