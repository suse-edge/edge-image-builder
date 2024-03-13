package build

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/cli/cmd"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/image/validation"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
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

// Runs the image definition validation, displaying the appropriate messages to the user in the event
// of a failure. Returns 'true' if the definition is valid; 'false' otherwise.
func isImageDefinitionValid(ctx *image.Context) bool {
	failedValidations := validation.ValidateDefinition(ctx)
	if len(failedValidations) == 0 {
		return true
	}

	log.Audit("Image definition validation found the following errors:")

	logMessageBuilder := strings.Builder{}

	orderedComponentNames := make([]string, 0, len(failedValidations))
	for c := range failedValidations {
		orderedComponentNames = append(orderedComponentNames, c)
	}
	slices.Sort(orderedComponentNames)

	for _, componentName := range orderedComponentNames {
		failures := failedValidations[componentName]
		log.Audit(fmt.Sprintf("  %s", componentName))
		for _, cf := range failures {
			log.Audit(fmt.Sprintf("    %s", cf.UserMessage))
			logMessageBuilder.WriteString(cf.UserMessage + "\n")
			if cf.Error != nil {
				logMessageBuilder.WriteString("\t" + cf.Error.Error() + "\n")
			}
		}
	}

	if s := logMessageBuilder.String(); s != "" {
		zap.S().Errorf("Image definition validation failures:\n%s", s)
	}

	log.AuditInfo(checkLogMessage)

	return false
}
