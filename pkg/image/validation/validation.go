package validation

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type FailedValidation struct {
	UserMessage string
	Error       error
}

type validateComponent func(ctx *image.Context) []FailedValidation

func ValidateDefinition(ctx *image.Context) map[string][]FailedValidation {
	failures := map[string][]FailedValidation{}

	schemaValidationFailure := validateAPIVersion(ctx)
	if schemaValidationFailure != nil {
		// If the schema is invalid, there's no reason to even attempt the rest of
		// the validation
		failures[apiVersionComponent] = []FailedValidation{*schemaValidationFailure}
		return failures
	}

	validations := map[string]validateComponent{
		imageComponent:    validateImage,
		osComponent:       validateOperatingSystem,
		registryComponent: validateEmbeddedArtifactRegistry,
		k8sComponent:      validateKubernetes,
	}
	for componentName, v := range validations {
		componentFailures := v(ctx)

		if len(componentFailures) > 0 {
			failures[componentName] = componentFailures
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
