package validation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	elementalComponent      = "Elemental"
	elementalConfigFilename = "elemental_config.yaml"
)

func validateElemental(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation

	elementalConfigDir := filepath.Join(ctx.ImageConfigDir, "elemental")
	if _, err := os.Stat(elementalConfigDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		failures = append(failures, FailedValidation{
			UserMessage: "Elemental config directory could not be read",
			Error:       err,
		})
		return failures
	}

	failures = append(failures, validateElementalDir(elementalConfigDir)...)

	if ctx.ImageDefinition.OperatingSystem.Packages.RegCode == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "Operating system package registration code field must be defined when using Elemental with SL Micro 6.0",
		})
	}

	return failures
}

func validateElementalDir(elementalConfigDir string) []FailedValidation {
	var failures []FailedValidation

	elementalConfigDirEntries, err := os.ReadDir(elementalConfigDir)
	if err != nil {
		failures = append(failures, FailedValidation{
			UserMessage: "Elemental config directory could not be read",
			Error:       err,
		})

		return failures
	}

	switch len(elementalConfigDirEntries) {
	case 0:
		failures = append(failures, FailedValidation{
			UserMessage: "Elemental config directory should not be present if it is empty",
		})
	case 1:
		if elementalConfigDirEntries[0].Name() != elementalConfigFilename {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Elemental config file should only be named `%s`", elementalConfigFilename),
			})
		}
	default:
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Elemental config directory should only contain a singular '%s' file", elementalConfigFilename),
		})
	}

	return failures
}
