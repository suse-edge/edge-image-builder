package build

import (
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/version"
	"github.com/urfave/cli/v2"
)

func Version(_ *cli.Context) error {
	log.Auditf("Edge Image Builder Version: %s", version.GetVersion())
	return nil
}
