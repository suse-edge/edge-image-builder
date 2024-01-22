package validation

import (
	"fmt"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	registryComponent = "Artifact Registry"
)

func validateEmbeddedArtifactRegistry(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation

	failures = append(failures, validateContainerImages(&ctx.ImageDefinition.EmbeddedArtifactRegistry)...)
	failures = append(failures, validateHelmCharts(&ctx.ImageDefinition.EmbeddedArtifactRegistry)...)

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

func validateHelmCharts(ear *image.EmbeddedArtifactRegistry) []FailedValidation {
	var failures []FailedValidation

	charts := ear.HelmCharts
	seenCharts := make(map[string]bool)
	for _, chart := range charts {
		if chart.Name == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'name' field is required for each entry in 'charts'.",
			})
		}

		if chart.RepoURL == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'repoURL' field is required for each entry in 'charts'.",
			})
		} else if !strings.HasPrefix(chart.RepoURL, "http") {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'repoURL' field must begin with either 'http://' or 'https://'.",
			})
		}

		if chart.Version == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'version' field is required for each entry in 'charts'.",
			})
		}

		if seenCharts[chart.Name] {
			msg := fmt.Sprintf("Duplicate chart name '%s' found in the 'charts' section.", chart.Name)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}
		seenCharts[chart.Name] = true
	}

	return failures
}
