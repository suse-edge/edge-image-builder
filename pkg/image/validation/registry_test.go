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
						URL: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
					{
						URL: "192.168.1.100:5000",
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
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "hello-world:latest",
					},
				},
			},
		},
		`url no credentials`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL:            "docker.io",
						Authentication: image.RegistryAuthentication{},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'username' field is required for each entry in 'embeddedArtifactRegistries.registries.credentials'.",
				"The 'password' field is required for each entry in 'embeddedArtifactRegistries.registries.credentials'.",
			},
		},
		`credentials missing username`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "",
							Password: "pass",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'username' field is required for each entry in 'embeddedArtifactRegistries.registries.credentials'.",
			},
		},
		`credentials missing password`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'password' field is required for each entry in 'embeddedArtifactRegistries.registries.credentials'.",
			},
		},
		`credentials duplicate URL`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
					{
						URL: "docker.io",
						Authentication: image.RegistryAuthentication{
							Username: "user2",
							Password: "pass2",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate URL 'docker.io' found in the 'embeddedArtifactRegistries.registries.url' section.",
			},
		},
		`invalid registry url`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name: "hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker...io",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
					{
						URL: "/docker.io/images",
						Authentication: image.RegistryAuthentication{
							Username: "user",
							Password: "pass",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Embedded artifact registry URL 'docker...io' could not be parsed.",
				"Embedded artifact registry URL '/docker.io/images' could not be parsed.",
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
