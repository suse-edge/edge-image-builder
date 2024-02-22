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
	groupsScriptName    = "13a-groups.sh"
	groupsComponentName = "groups"
)

//go:embed templates/13a-add-groups.sh.tpl
var groupsScript string

func configureGroups(ctx *image.Context) ([]string, error) {
	// Punch out early if there are no groups
	if len(ctx.ImageDefinition.OperatingSystem.Groups) == 0 {
		log.AuditComponentSkipped(groupsComponentName)
		return nil, nil
	}

	data, err := template.Parse(groupsScriptName, groupsScript, ctx.ImageDefinition.OperatingSystem.Groups)
	if err != nil {
		log.AuditComponentFailed(groupsComponentName)
		return nil, fmt.Errorf("parsing the group script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, groupsScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		log.AuditComponentFailed(groupsComponentName)
		return nil, fmt.Errorf("writing %s to the combustion directory: %w", groupsScriptName, err)
	}

	log.AuditComponentSuccessful(groupsComponentName)
	return []string{groupsScriptName}, nil
}
