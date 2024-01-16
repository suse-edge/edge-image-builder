package validation

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type FailedValidation struct {
	component   string
	userMessage string
	err         error
}

type validateComponent func(ctx *image.Context) []FailedValidation

func ValidateDefinition(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation

	validations := []validateComponent{
		validateImage,
	}
	for _, v := range validations {
		componentFailures := v(ctx)
		failures = append(failures, componentFailures...)
	}

	return failures
}
