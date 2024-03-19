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
)

const (
	checkValidationLogMessage = "Please check the log file under the validation directory for more information."
)

func Validate(_ *cli.Context) error {
	args := &cmd.BuildArgs

	validationDir := filepath.Join(args.ConfigDir, "_validation")
	if err := os.MkdirAll(validationDir, os.ModePerm); err != nil {
		log.Auditf("The validation directory could not be setup under the configuration directory '%s'.", args.ConfigDir)
		return err
	}

	// This needs to occur as early as possible so that the subsequent calls can use the log
	timestamp := time.Now().Format("Jan02_15-04-05")
	logFilename := filepath.Join(validationDir, fmt.Sprintf("eib-validate-%s.log", timestamp))
	log.ConfigureGlobalLogger(logFilename)

	log.AuditInfo("Checking image config dir...")

	if err := imageConfigDirExists(args.ConfigDir); err != nil {
		cmd.LogError(err, checkValidationLogMessage)
		os.Exit(1)
	}

	log.AuditInfo("Parsing image definition...")

	imageDefinition, err := parseImageDefinition(args.ConfigDir, args.DefinitionFile)
	if err != nil {
		cmd.LogError(err, checkValidationLogMessage)
		os.Exit(1)
	}

	ctx := &image.Context{
		ImageConfigDir:  args.ConfigDir,
		ImageDefinition: imageDefinition,
	}

	log.AuditInfo("Validating image definition...")

	if err = validateImageDefinition(ctx); err != nil {
		cmd.LogError(err, checkValidationLogMessage)
		os.Exit(1)
	}

	log.AuditInfo("The specified image definition is valid.")

	return nil
}

func validateImageDefinition(ctx *image.Context) *cmd.Error {
	failedValidations := validation.ValidateDefinition(ctx)
	if len(failedValidations) == 0 {
		return nil
	}

	logMessageBuilder := strings.Builder{}
	userMessageBuilder := strings.Builder{}

	userMessageBuilder.WriteString("Image definition validation found the following errors:\n")
	logMessageBuilder.WriteString("Image definition validation failures:\n")

	orderedComponentNames := make([]string, 0, len(failedValidations))
	for c := range failedValidations {
		orderedComponentNames = append(orderedComponentNames, c)
	}
	slices.Sort(orderedComponentNames)

	for _, componentName := range orderedComponentNames {
		userMessageBuilder.WriteString("  " + componentName + "\n")

		for _, cf := range failedValidations[componentName] {
			userMessageBuilder.WriteString("    " + cf.UserMessage + "\n")
			logMessageBuilder.WriteString("  " + cf.UserMessage + "\n")
			if cf.Error != nil {
				logMessageBuilder.WriteString("    " + cf.Error.Error() + "\n")
			}
		}
	}

	return &cmd.Error{
		UserMessage: userMessageBuilder.String(),
		LogMessage:  logMessageBuilder.String(),
	}
}
