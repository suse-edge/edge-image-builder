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
	timeComponentName = "time"
	timeScriptName    = "11-time-setup.sh"
)

//go:embed templates/11-time-setup.sh.tpl
var timeScript string

func configureTime(ctx *image.Context) ([]string, error) {
	time := ctx.ImageDefinition.OperatingSystem.Time
	if time.Timezone == "" {
		log.AuditComponentSkipped(timeComponentName)
		return nil, nil
	}

	if err := writeTimeCombustionScript(ctx); err != nil {
		log.AuditComponentFailed(timeComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(timeComponentName)
	return []string{timeScriptName}, nil
}

func writeTimeCombustionScript(ctx *image.Context) error {
	timeScriptFilename := filepath.Join(ctx.CombustionDir, timeScriptName)

	values := struct {
		Timezone  string
		Pools     []string
		Servers   []string
		ForceWait bool
	}{
		Timezone:  ctx.ImageDefinition.OperatingSystem.Time.Timezone,
		Pools:     ctx.ImageDefinition.OperatingSystem.Time.NtpConfiguration.Pools,
		Servers:   ctx.ImageDefinition.OperatingSystem.Time.NtpConfiguration.Servers,
		ForceWait: ctx.ImageDefinition.OperatingSystem.Time.NtpConfiguration.ForceWait,
	}

	data, err := template.Parse(timeScriptName, timeScript, values)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", timeScriptName, err)
	}

	if err := os.WriteFile(timeScriptFilename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", timeScriptFilename, err)
	}
	return nil
}
