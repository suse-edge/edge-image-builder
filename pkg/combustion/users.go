package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	usersScriptName    = "13b-add-users.sh"
	usersComponentName = "users"
)

//go:embed templates/13b-add-users.sh.tpl
var usersScript string

func configureUsers(ctx *image.Context) ([]string, error) {
	// Punch out early if there are no users
	if len(ctx.ImageDefinition.OperatingSystem.Users) == 0 {
		log.AuditComponentSkipped(usersComponentName)
		return nil, nil
	}

	data, err := template.Parse(usersScriptName, usersScript, ctx.ImageDefinition.OperatingSystem.Users)
	if err != nil {
		log.AuditComponentFailed(usersComponentName)
		return nil, fmt.Errorf("parsing users script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, usersScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		log.AuditComponentFailed(usersComponentName)
		return nil, fmt.Errorf("writing %s to the combustion directory: %w", usersScriptName, err)
	}

	log.AuditComponentSuccessful(usersComponentName)
	return []string{usersScriptName}, nil
}
