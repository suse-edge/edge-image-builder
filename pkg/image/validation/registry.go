package validation

import (
	"fmt"

	"github.com/containers/image/v5/docker/reference"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	registryComponent = "Artifact Registry"
)

func validateEmbeddedArtifactRegistry(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation

	if len(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages) > 0 {
		failures = append(failures, validateContainerImages(&ctx.ImageDefinition.EmbeddedArtifactRegistry)...)
		failures = append(failures, validateRegistries(&ctx.ImageDefinition.EmbeddedArtifactRegistry)...)
	}

	return failures
}

func validateContainerImages(ear *image.EmbeddedArtifactRegistry) []FailedValidation {
	var failures []FailedValidation

	seenContainerImages := make(map[string]bool)
	for _, cImage := range ear.ContainerImages {
		if cImage.Name == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'name' field is required for each entry in 'images'.",
			})
		}

		if seenContainerImages[cImage.Name] {
			msg := fmt.Sprintf("Duplicate image name '%s' found in the 'images' section.", cImage.Name)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}
		seenContainerImages[cImage.Name] = true
	}

	return failures
}

func validateRegistries(ear *image.EmbeddedArtifactRegistry) []FailedValidation {
	var failures []FailedValidation

	failures = append(failures, validateURLs(ear)...)
	failures = append(failures, validateCredentials(ear)...)

	return failures
}

func validateURLs(ear *image.EmbeddedArtifactRegistry) []FailedValidation {
	var failures []FailedValidation

	seenRegistryURLs := make(map[string]bool)
	for _, registry := range ear.Registries {
		if registry.URL == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'url' field is required for each entry in 'embeddedArtifactRegistries.registries'.",
			})
		}

		_, err := reference.Parse(registry.URL)
		if err != nil {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Embedded artifact registry URL '%s' could not be parsed.", registry.URL),
				Error:       err,
			})
		}

		if seenRegistryURLs[registry.URL] {
			msg := fmt.Sprintf("Duplicate URL '%s' found in the 'embeddedArtifactRegistries.registries.url' section.", registry.URL)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}

		seenRegistryURLs[registry.URL] = true
	}

	return failures
}

func validateCredentials(ear *image.EmbeddedArtifactRegistry) []FailedValidation {
	var failures []FailedValidation

	for _, registry := range ear.Registries {
		if registry.Authentication.Username == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'username' field is required for each entry in 'embeddedArtifactRegistries.registries.credentials'.",
			})
		}

		if registry.Authentication.Password == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'password' field is required for each entry in 'embeddedArtifactRegistries.registries.credentials'.",
			})
		}
	}

	return failures
}
