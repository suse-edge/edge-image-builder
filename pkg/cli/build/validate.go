package build

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/cli/cmd"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/urfave/cli/v2"
)

const (
	validateLogFile = "eib-validate.log"
)

func Validate(_ *cli.Context) error {
	args := &cmd.BuildArgs

	timestamp := time.Now().Format("Jan02_15-04-05")
	validationDir := filepath.Join(args.ConfigDir, fmt.Sprintf("validate-%s", timestamp))
	if err := os.MkdirAll(validationDir, os.ModePerm); err != nil {
		log.Auditf("The validation directory could not be setup under the configuration directory '%s'.", args.ConfigDir)
		return err
	}

	// This needs to occur as early as possible so that the subsequent calls can use the log
	log.ConfigureGlobalLogger(filepath.Join(validationDir, validateLogFile))

	if !imageConfigDirExists(args.ConfigDir) {
		os.Exit(1)
	}

	imageDefinition := parseImageDefinition(args.ConfigDir, args.DefinitionFile)
	if imageDefinition == nil {
		os.Exit(1)
	}

	ctx := &image.Context{
		ImageConfigDir:  args.ConfigDir,
		ImageDefinition: imageDefinition,
	}

	if !isImageDefinitionValid(ctx) {
		os.Exit(1)
	}

	log.AuditInfo("The specified image definition is valid.")

	return nil
}
