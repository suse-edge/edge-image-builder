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
				"Field `kubernetes.helm.charts.apiVersions` is only available in API version >= 1.1",
				"Field `operatingSystem.enableFIPS` is only available in API version >= 1.1",
				"Field `kubernetes.network.apiVIP6` is only available in API version >= 1.2",
				"Field `kubernetes.helm.charts.releaseName` is only available in API version >= 1.2",
				"Field `operatingSystem.rawConfiguration.luksKey` is only available in API version >= 1.2",
				"Field `operatingSystem.rawConfiguration.expandEncryptedPartition` is only available in API version >= 1.2",
				"Field `operatingSystem.packages.enableExtras` is only available in API version >= 1.2",
				"Field `embeddedArtifactRegistry.registries` is only available in API version >= 1.2",
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
				"Field `kubernetes.network.apiVIP6` is only available in API version >= 1.2",
				"Field `kubernetes.helm.charts.releaseName` is only available in API version >= 1.2",
				"Field `operatingSystem.rawConfiguration.luksKey` is only available in API version >= 1.2",
				"Field `operatingSystem.rawConfiguration.expandEncryptedPartition` is only available in API version >= 1.2",
				"Field `operatingSystem.packages.enableExtras` is only available in API version >= 1.2",
				"Field `embeddedArtifactRegistry.registries` is only available in API version >= 1.2",
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
		`valid new fields for 1.3`: {
			ImageDefinition: image.Definition{
				APIVersion: "1.3",
				OperatingSystem: image.OperatingSystem{
					Packages: image.Packages{
						AdditionalRepos: []image.AddRepo{
							{
								URL:      "https://developer.download.nvidia.com/compute/cuda/repos/sles15/x86_64/",
								Priority: 50,
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
