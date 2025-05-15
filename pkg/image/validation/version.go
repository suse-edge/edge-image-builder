package validation

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	versionComponent = "Version"
)

// Note: This method of validating the EIB version in the image definition is only a temporary implementation
// until a more robust solution can be found.
func validateVersion(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation
	definition := *ctx.ImageDefinition

	var apiVersionsDefined bool
	for i := range definition.Kubernetes.Helm.Charts {
		if len(definition.Kubernetes.Helm.Charts[i].APIVersions) != 0 {
			apiVersionsDefined = true
		}
	}

	if definition.APIVersion == "1.0" {
		if apiVersionsDefined {
			failures = append(failures, FailedValidation{
				UserMessage: "Kubernetes field `helm.charts.apiVersions` is only available in EIB version >= 1.1",
			})

			if definition.OperatingSystem.EnableFIPS {
				failures = append(failures, FailedValidation{
					UserMessage: "Operating system field `enableFIPS` is only available in EIB version >= 1.1",
				})
			}
		}
	}

	if definition.APIVersion != "1.2" {
		if definition.Kubernetes.Network.APIVIP6 != "" {
			failures = append(failures, FailedValidation{
				UserMessage: "Kubernetes field `network.apiVIP6` is only available in EIB version >= 1.2",
			})
		}

		for _, charts := range definition.Kubernetes.Helm.Charts {
			if charts.ReleaseName != "" {
				failures = append(failures, FailedValidation{
					UserMessage: "Kubernetes field `helm.charts.releaseName` is only available in EIB version >= 1.2",
				})

				break
			}
		}

		if definition.OperatingSystem.RawConfiguration.LUKSKey != "" {
			failures = append(failures, FailedValidation{
				UserMessage: "Operating system field `rawConfiguration.luksKey` is only available in EIB version >= 1.2",
			})
		}

		if definition.OperatingSystem.RawConfiguration.ExpandEncryptedPartition {
			failures = append(failures, FailedValidation{
				UserMessage: "Operating system field `rawConfiguration.expandEncryptedPartition` is only available in EIB version >= 1.2",
			})
		}

		if definition.OperatingSystem.Packages.EnableExtras {
			failures = append(failures, FailedValidation{
				UserMessage: "Operating system field `packages.enableExtras` is only available in EIB version >= 1.2",
			})
		}

		if len(definition.EmbeddedArtifactRegistry.Registries) != 0 {
			failures = append(failures, FailedValidation{
				UserMessage: "Embedded artifact registry field `registries` is only available in EIB version >= 1.2",
			})
		}
	}

	return failures
}
