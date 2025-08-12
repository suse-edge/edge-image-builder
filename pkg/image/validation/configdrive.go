package validation

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	configDriveComponent = "configdrive"
)

func validateConfigDrive(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation

	if !ctx.IsConfigDrive {
		return failures
	}

	def := ctx.ImageDefinition
	failures = append(failures, validateImageSection(def)...)
	failures = append(failures, validateOperatingSystemSection(def)...)

	return failures
}

func validateImageSection(def *image.Definition) []FailedValidation {
	var failures []FailedValidation

	if def.Image.OutputImageName != "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'outputImageName' field is not valid for generating config drives. The name of the output " +
				"file should be defined through the '--output' argument.",
		})
	}

	if def.Image.ImageType != "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'imageType' field is not valid for generating config drives. The output type " +
				"should be defined through the '--output-type' argument.",
		})
	}

	if def.Image.Arch != "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'arch' field is not valid for generating config drives. The architecture of the generated " +
				"config drive should be defined through the '--arch' argument.",
		})
	}

	if def.Image.BaseImage != "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'baseImage' field is not valid for generating config drives.",
		})
	}

	return failures
}

func validateOperatingSystemSection(def *image.Definition) []FailedValidation {
	var failures []FailedValidation
	emptyRawConfig := image.RawConfiguration{}

	if def.OperatingSystem.RawConfiguration != emptyRawConfig {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'operatingSystem.rawConfiguration' field is not valid for generating config drives.",
		})
	}

	if def.OperatingSystem.IsoConfiguration.InstallDevice != "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'operatingSystem.isoConfiguration' field is not valid for generating config drives.",
		})
	}

	if def.OperatingSystem.EnableFIPS {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'operatingSystem.enableFIPS' field is not valid for generating config drives.",
		})
	}

	if len(def.OperatingSystem.Packages.PKGList) != 0 || def.OperatingSystem.Packages.EnableExtras ||
		def.OperatingSystem.Packages.RegCode != "" || len(def.OperatingSystem.Packages.AdditionalRepos) != 0 ||
		def.OperatingSystem.Packages.NoGPGCheck {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'operatingSystem.packages' field is not valid for generating config drives.",
		})
	}

	if len(def.OperatingSystem.KernelArgs) != 0 {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'operatingSystem.kernelArgs' field is not valid for generating config drives.",
		})
	}

	return failures
}
