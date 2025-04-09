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
						Name: "foo",
					},
				},
				Registries: []image.Registry{
					{
						URI: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
					{
						URI: "192.168.1.100:5000",
						Authentication: image.RegistryAuthentication{
							Username: "user2",
							Password: "pass2",
						},
					},
				},
			},
		},
		`image definition failure`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "", // trips the missing name validation
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'name' field is required for each entry in 'images'.",
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

func TestValidateRegistries(t *testing.T) {
	tests := map[string]struct {
		Registry               image.EmbeddedArtifactRegistry
		ExpectedFailedMessages []string
	}{
		`no authentication`: {
			Registry: image.EmbeddedArtifactRegistry{},
		},
		`URI no credentials`: {
			Registry: image.EmbeddedArtifactRegistry{
				Registries: []image.Registry{
					{
						URI:            "docker.io",
						Authentication: image.RegistryAuthentication{},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'username' field is required for each entry in 'embeddedArtifactRegistry.registries.credentials'.",
				"The 'password' field is required for each entry in 'embeddedArtifactRegistry.registries.credentials'.",
			},
		},
		`credentials missing username`: {
			Registry: image.EmbeddedArtifactRegistry{
				Registries: []image.Registry{
					{
						URI: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "",
							Password: "pass",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'username' field is required for each entry in 'embeddedArtifactRegistry.registries.credentials'.",
			},
		},
		`credentials missing password`: {
			Registry: image.EmbeddedArtifactRegistry{
				Registries: []image.Registry{
					{
						URI: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'password' field is required for each entry in 'embeddedArtifactRegistry.registries.credentials'.",
			},
		},
		`credentials duplicate URI`: {
			Registry: image.EmbeddedArtifactRegistry{
				Registries: []image.Registry{
					{
						URI: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
					{
						URI: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "user2",
							Password: "pass2",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate registry URI 'docker.io' found in the 'embeddedArtifactRegistry.registries' section.",
			},
		},
		`invalid registry URI`: {
			Registry: image.EmbeddedArtifactRegistry{
				Registries: []image.Registry{
					{
						URI: "docker...io",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
					{
						URI: "/docker.io/images",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
					{
						URI: "https://docker.io/images",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Embedded artifact registry URI 'docker...io' could not be parsed.",
				"Embedded artifact registry URI '/docker.io/images' could not be parsed.",
				"Embedded artifact registry URI 'https://docker.io/images' could not be parsed.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ear := test.Registry
			failures := validateRegistries(&ear)
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
