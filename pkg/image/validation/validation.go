package validation

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type FailedValidation struct {
	Component   string
	UserMessage string
	Error       error
}

type validateComponent func(ctx *image.Context) []FailedValidation

func ValidateDefinition(ctx *image.Context) map[string][]FailedValidation {
	failures := map[string][]FailedValidation{}

	validations := []validateComponent{
		validateImage,
		validateOperatingSystem,
		validateEmbeddedArtifactRegistry,
		validateKubernetes,
	}
	for _, v := range validations {
		componentFailures := v(ctx)

		if len(componentFailures) > 0 {
			failures[componentFailures[0].Component] = componentFailures
		}
	}

	return failures
}

func findDuplicates(items []string) []string {
	var duplicates []string

	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item] {
			duplicates = append(duplicates, item)
		}
		seen[item] = true
	}

	return duplicates
}
