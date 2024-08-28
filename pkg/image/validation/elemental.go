package validation

import (
	"errors"
	"fmt"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

const (
	elementalComponent = "Elemental"
)

func validateElemental(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation

	elementalConfigDir := filepath.Join(ctx.ImageConfigDir, "elemental")
	failures = append(failures, validateElementalDir(elementalConfigDir)...)

	if ctx.ImageDefinition.APIVersion == "1.1" && ctx.ImageDefinition.OperatingSystem.Packages.RegCode == "" {
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
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		zap.S().Errorf("Elemental config directory could not be read: %s", err)
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Elemental config directory could not be read: %s", err),
		})
	}

	if len(elementalConfigDirEntries) == 0 {
		failures = append(failures, FailedValidation{
			UserMessage: "Elemental config directory should not be present if it is empty",
		})
	}

	if len(elementalConfigDirEntries) > 1 {
		failures = append(failures, FailedValidation{
			UserMessage: "Elemental config directory should only contain a singular 'elemental_config.yaml' file",
		})
	}

	if len(elementalConfigDirEntries) == 1 {
		if elementalConfigDirEntries[0].Name() != "elemental_config.yaml" {
			failures = append(failures, FailedValidation{
				UserMessage: "Elemental config file should only be named `elemental_config.yaml`",
			})
		}
	}

	return failures
}
