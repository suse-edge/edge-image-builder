package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

const (
	timesyncScriptName    = "15-wait-for-timesync.sh"
	timesyncComponentName = "timesync"
)

//go:embed templates/15-wait-for-timesync.sh
var timesyncScript string

func configureTimesync(ctx *image.Context) ([]string, error) {
	ntpConfig := ctx.ImageDefinition.OperatingSystem.Time.NtpConfiguration
	if !ntpConfig.ForceWait {
		log.AuditComponentSkipped(timesyncComponentName)
		return nil, nil
	}

	filename := filepath.Join(ctx.CombustionDir, timesyncScriptName)

	if err := os.WriteFile(filename, []byte(timesyncScript), fileio.ExecutablePerms); err != nil {
		log.AuditComponentFailed(timesyncComponentName)
		return nil, fmt.Errorf("copying script %s: %w", timesyncScriptName, err)
	}

	log.AuditComponentSuccessful(timesyncComponentName)
	return []string{timesyncScriptName}, nil
}
