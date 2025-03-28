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
						Credentials: []image.RegistryAuthentication{
							{
								Username: "user",
								Password: "pass",
							},
							{
								Username: "user2",
								Password: "pass2",
							},
						},
					},
					{
						URL: "192.168.1.100:5000",
						Credentials: []image.RegistryAuthentication{
							{
								Username: "user",
								Password: "pass",
							},
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
						"hello-world:latest",
					},
				},
			},
		},
		`url no credentials`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						"hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL:         "docker.io",
						Credentials: nil,
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'credentials' field is required for 'embeddedArtifactRegistries.registries' entry 'docker.io'.",
			},
		},
		`credentials missing username`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						"hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker.io",
						Credentials: []image.RegistryAuthentication{
							{
								Username: "",
								Password: "pass",
							},
							{
								Username: "user2",
								Password: "pass2",
							},
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
						"hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker.io",
						Credentials: []image.RegistryAuthentication{
							{
								Username: "user",
								Password: "",
							},
							{
								Username: "user2",
								Password: "pass2",
							},
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
						"hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker.io",
						Credentials: []image.RegistryAuthentication{
							{
								Username: "user",
								Password: "pass",
							},
						},
					},
					{
						URL: "docker.io",
						Credentials: []image.RegistryAuthentication{
							{
								Username: "user2",
								Password: "pass2",
							},
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate URL 'docker.io' found in the 'embeddedArtifactRegistries.registries.url' section.",
			},
		},
		`credentials duplicate username`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						"hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker.io",
						Credentials: []image.RegistryAuthentication{
							{
								Username: "user",
								Password: "pass",
							},
							{
								Username: "user",
								Password: "pass2",
							},
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate username 'user' found for registry 'docker.io' in the 'embeddedArtifactRegistries.registries.credentials.username' section.",
			},
		},
		`invalid registry url`: {
			Registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						"hello-world:latest",
					},
				},
				Registries: []image.Registry{
					{
						URL: "docker...io",
						Credentials: []image.RegistryAuthentication{
							{
								Username: "user",
								Password: "pass",
							},
						},
					},
					{
						URL: "/docker.io/images",
						Credentials: []image.RegistryAuthentication{
							{
								Username: "user",
								Password: "pass",
							},
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
