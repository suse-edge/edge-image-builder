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
	systemdComponentName = "systemd"
	systemdScriptName    = "20-systemd.sh"
)

//go:embed templates/20-systemd.sh.tpl
var systemdTemplate string

func configureSystemd(ctx *image.Context) ([]string, error) {
	// Nothing to do if both lists are empty
	systemd := ctx.ImageDefinition.OperatingSystem.Systemd
	if len(systemd.Enable) == 0 && len(systemd.Disable) == 0 {
		log.AuditComponentSkipped(systemdComponentName)
		return nil, nil
	}

	data, err := template.Parse(systemdScriptName, systemdTemplate, ctx.ImageDefinition.OperatingSystem.Systemd)
	if err != nil {
		log.AuditComponentFailed(systemdComponentName)
		return nil, fmt.Errorf("applying systemd script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, systemdScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		log.AuditComponentFailed(systemdComponentName)
		return nil, fmt.Errorf("writing systemd combustion file: %w", err)
	}

	log.AuditComponentSuccessful(systemdComponentName)
	return []string{systemdScriptName}, nil
}
