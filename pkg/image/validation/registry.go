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

	failures = append(failures, validateRegistries(&ctx.ImageDefinition.EmbeddedArtifactRegistry)...)
	failures = append(failures, validateContainerImages(&ctx.ImageDefinition.EmbeddedArtifactRegistry)...)

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
		if registry.URI == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'uri' field is required for each entry in 'embeddedArtifactRegistry.registries'.",
			})
		}

		_, err := reference.Parse(registry.URI)
		if err != nil {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Embedded artifact registry URI '%s' could not be parsed.", registry.URI),
				Error:       err,
			})

			continue
		}

		if seenRegistryURLs[registry.URI] {
			msg := fmt.Sprintf("Duplicate registry URI '%s' found in the 'embeddedArtifactRegistry.registries' section.", registry.URI)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}

		seenRegistryURLs[registry.URI] = true
	}

	return failures
}

func validateCredentials(ear *image.EmbeddedArtifactRegistry) []FailedValidation {
	var failures []FailedValidation

	for _, registry := range ear.Registries {
		if registry.Authentication.Username == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'username' field is required for each entry in 'embeddedArtifactRegistry.registries.credentials'.",
			})
		}

		if registry.Authentication.Password == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'password' field is required for each entry in 'embeddedArtifactRegistry.registries.credentials'.",
			})
		}
	}

	return failures
}
