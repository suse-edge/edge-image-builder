package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestValidateVersion(t *testing.T) {
	tests := map[string]struct {
		ImageDefinition        image.Definition
		ExpectedFailedMessages []string
	}{
		`invalid 1.0 definition`: {
			ImageDefinition: image.Definition{
				APIVersion: "1.0",
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						LUKSKey:                  "1234",
						ExpandEncryptedPartition: true,
					},
					EnableFIPS: true,
					Packages: image.Packages{
						EnableExtras: true,
					},
				},
				Kubernetes: image.Kubernetes{
					Network: image.Network{
						APIVIP6: "fd12:3456:789a::21",
					},
					Helm: image.Helm{
						Charts: []image.HelmChart{
							{
								APIVersions: []string{"1.30.3+k3s1"},
								ReleaseName: "release-1",
							},
						},
					},
				},
				EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
					Registries: []image.Registry{
						{
							URI: "docker.io",
							Authentication: image.RegistryAuthentication{
								Username: "user",
								Password: "pass",
							},
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Kubernetes field `helm.charts.apiVersions` is only available in EIB version >= 1.1",
				"Operating system field `enableFIPS` is only available in EIB version >= 1.1",
				"Kubernetes field `network.apiVIP6` is only available in EIB version >= 1.2",
				"Kubernetes field `helm.charts.releaseName` is only available in EIB version >= 1.2",
				"Operating system field `rawConfiguration.luksKey` is only available in EIB version >= 1.2",
				"Operating system field `rawConfiguration.expandEncryptedPartition` is only available in EIB version >= 1.2",
				"Operating system field `packages.enableExtras` is only available in EIB version >= 1.2",
				"Embedded artifact registry field `registries` is only available in EIB version >= 1.2",
			},
		},
		`invalid 1.1 definition`: {
			ImageDefinition: image.Definition{
				APIVersion: "1.1",
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						LUKSKey:                  "1234",
						ExpandEncryptedPartition: true,
					},
					Packages: image.Packages{
						EnableExtras: true,
					},
				},
				Kubernetes: image.Kubernetes{
					Network: image.Network{
						APIVIP6: "fd12:3456:789a::21",
					},
					Helm: image.Helm{
						Charts: []image.HelmChart{
							{
								ReleaseName: "release-1",
							},
						},
					},
				},
				EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
					Registries: []image.Registry{
						{
							URI: "docker.io",
							Authentication: image.RegistryAuthentication{
								Username: "user",
								Password: "pass",
							},
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Kubernetes field `network.apiVIP6` is only available in EIB version >= 1.2",
				"Kubernetes field `helm.charts.releaseName` is only available in EIB version >= 1.2",
				"Operating system field `rawConfiguration.luksKey` is only available in EIB version >= 1.2",
				"Operating system field `rawConfiguration.expandEncryptedPartition` is only available in EIB version >= 1.2",
				"Operating system field `packages.enableExtras` is only available in EIB version >= 1.2",
				"Embedded artifact registry field `registries` is only available in EIB version >= 1.2",
			},
		},
		`valid new fields for 1.1`: {
			ImageDefinition: image.Definition{
				APIVersion:      "1.1",
				OperatingSystem: image.OperatingSystem{EnableFIPS: true},
				Kubernetes: image.Kubernetes{
					Helm: image.Helm{Charts: []image.HelmChart{{APIVersions: []string{"1.30.3+k3s1"}}}},
				},
			},
		},
		`valid new fields for 1.2`: {
			ImageDefinition: image.Definition{
				APIVersion: "1.2",
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						LUKSKey:                  "1234",
						ExpandEncryptedPartition: true,
					},
					Packages: image.Packages{
						EnableExtras: true,
					},
				},
				Kubernetes: image.Kubernetes{
					Network: image.Network{
						APIVIP6: "fd12:3456:789a::21",
					},
					Helm: image.Helm{
						Charts: []image.HelmChart{
							{
								ReleaseName: "release-1",
							},
						},
					},
				},
				EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
					Registries: []image.Registry{
						{
							URI: "docker.io",
							Authentication: image.RegistryAuthentication{
								Username: "user",
								Password: "pass",
							},
						},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			imageDef := test.ImageDefinition
			ctx := image.Context{
				ImageDefinition: &imageDef,
			}
			failedValidations := validateVersion(&ctx)
			assert.Len(t, failedValidations, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failedValidations {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}
