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
		`valid version with Helm APIVersions`: {
			ImageDefinition: image.Definition{
				APIVersion: "1.1",
				Kubernetes: image.Kubernetes{Helm: image.Helm{Charts: []image.HelmChart{
					{
						APIVersions: []string{"1.30.3+k3s1"},
					},
				}}},
			},
		},
		`invalid version with Helm APIVersions`: {
			ImageDefinition: image.Definition{
				APIVersion: "1.0",
				Kubernetes: image.Kubernetes{Helm: image.Helm{Charts: []image.HelmChart{
					{
						APIVersions: []string{"1.30.3+k3s1"},
					},
				}}},
			},
			ExpectedFailedMessages: []string{
				"Helm chart APIVersions field is not supported in EIB version 1.0, must use EIB version 1.1",
			},
		},
		`invalid version with FIPS enabled`: {
			ImageDefinition: image.Definition{
				APIVersion: "1.0",
				OperatingSystem: image.OperatingSystem{
					EnableFIPS: true,
				},
			},
			ExpectedFailedMessages: []string{
				"Automated FIPS configuration is not supported in EIB version 1.0, please use EIB version >= 1.1",
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
