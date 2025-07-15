package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/combustion"
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

	if ctx.ImageDefinition.Image.ImageType != image.TypeCombustionIso && ctx.ImageDefinition.Image.ImageType != image.TypeTar {
		failures = append(failures, validateElementalConfiguration(ctx)...)
	}

	failures = append(failures, validateElementalDir(elementalConfigDir)...)

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

func validateElementalConfiguration(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation

	rpmDirEntries, err := os.ReadDir(combustion.RPMsPath(ctx))
	if err != nil && !os.IsNotExist(err) {
		failures = append(failures, FailedValidation{
			UserMessage: "RPM directory could not be read",
			Error:       err,
		})
	}

	var foundPackages []string
	var notFoundPackages []string
	for _, pkg := range combustion.ElementalPackages {
		if slices.ContainsFunc(rpmDirEntries, func(entry os.DirEntry) bool {
			return strings.Contains(entry.Name(), pkg)
		}) {
			foundPackages = append(foundPackages, pkg)
		} else {
			notFoundPackages = append(notFoundPackages, pkg)
		}
	}

	if len(foundPackages) == 0 {
		if ctx.ImageDefinition.OperatingSystem.Packages.RegCode == "" {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Operating system package registration code field must be defined when using Elemental "+
					"or the %s RPMs must be manually side-loaded", combustion.ElementalPackages),
			})
		}
	} else if len(foundPackages) != len(combustion.ElementalPackages) {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Not all of the necessary Elemental packages are provided, packages found: %s, packages missing: %s", foundPackages, notFoundPackages),
		})
	}

	return failures
}
