package build

import (
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"github.com/urfave/cli/v2"
)

func Version(_ *cli.Context) error {
	log.AuditInfof("Edge Image Builder Version: %s", template.GetVersion())
	return nil
}
