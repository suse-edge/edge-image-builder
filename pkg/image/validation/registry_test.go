package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestValidateEmbeddedArtifactRegistry(t *testing.T) {
	tests := map[string]struct {
		Registry               image.EmbeddedArtifactRegistry
		ExpectedFailedMessages []string
	}{
		`no registry`: {
			Registry: image.EmbeddedArtifactRegistry{},
		},
		`full valid example`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name:           "foo",
						SupplyChainKey: "key",
					},
				},
				HelmCharts: []image.HelmChart{
					{
						Name:    "bar",
						RepoURL: "http://bar.com",
						Version: "3.14",
					},
				},
			},
		},
		`failures in both sections`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "", // trips the missing name validation
					},
				},
				HelmCharts: []image.HelmChart{
					{
						Name:    "", // trips the missing name validation
						RepoURL: "http://doesntmatter.com",
						Version: "31",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'name' field is required for each entry in 'images'.",
				"The 'name' field is required for each entry in 'charts'.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ear := test.Registry
			ctx := image.Context{
				ImageDefinition: &image.Definition{
					EmbeddedArtifactRegistry: ear,
				},
			}
			failures := validateEmbeddedArtifactRegistry(&ctx)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestValidateContainerImages(t *testing.T) {
	tests := map[string]struct {
		Registry               image.EmbeddedArtifactRegistry
		ExpectedFailedMessages []string
	}{
		`no images`: {
			Registry: image.EmbeddedArtifactRegistry{},
		},
		`missing name`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "valid",
					},
					{
						Name: "",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'name' field is required for each entry in 'images'.",
			},
		},
		`duplicate name`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "foo",
					},
					{
						Name: "bar",
					},
					{
						Name: "foo",
					},
					{
						Name: "baz",
					},
					{
						Name: "bar",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate image name 'foo' found in the 'images' section.",
				"Duplicate image name 'bar' found in the 'images' section.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ear := test.Registry
			failures := validateContainerImages(&ear)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestValidateHelmCharts(t *testing.T) {
	tests := map[string]struct {
		Registry               image.EmbeddedArtifactRegistry
		ExpectedFailedMessages []string
	}{
		`no helm charts`: {
			Registry: image.EmbeddedArtifactRegistry{},
		},
		`valid charts`: {
			Registry: image.EmbeddedArtifactRegistry{
				HelmCharts: []image.HelmChart{
					{
						Name:    "foo",
						RepoURL: "http://valid.com", // shows http:// is allowed
						Version: "1.0",
					},
					{
						Name:    "bar",
						RepoURL: "https://valid.com", // shows https:// is allowed
						Version: "2.0",
					},
				},
			},
		},
		`missing fields`: {
			Registry: image.EmbeddedArtifactRegistry{
				HelmCharts: []image.HelmChart{
					{},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'name' field is required for each entry in 'charts'.",
				"The 'repoURL' field is required for each entry in 'charts'.",
				"The 'version' field is required for each entry in 'charts'.",
			},
		},
		`duplicate chart`: {
			Registry: image.EmbeddedArtifactRegistry{
				HelmCharts: []image.HelmChart{
					{
						Name:    "foo",
						RepoURL: "http://foo.com",
						Version: "1.0",
					},
					{
						Name:    "foo",
						RepoURL: "https://bar.com",
						Version: "2.0",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate chart name 'foo' found in the 'charts' section.",
			},
		},
		`invalid repo`: {
			Registry: image.EmbeddedArtifactRegistry{
				HelmCharts: []image.HelmChart{
					{
						Name:    "foo",
						RepoURL: "example.com",
						Version: "1.0",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'repoURL' field must begin with either 'http://' or 'https://'.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ear := test.Registry
			failures := validateHelmCharts(&ear)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}
