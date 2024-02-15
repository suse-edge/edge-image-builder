package validation

import "github.com/suse-edge/edge-image-builder/pkg/image"

const (
	apiVersionComponent = "Definition Schema"
)

func validateAPIVersion(ctx *image.Context) *FailedValidation {
	definitionVersion := ctx.ImageDefinition.APIVersion

	if definitionVersion != "1.0" {
		return &FailedValidation{
			UserMessage: "This version of Edge Image Builder only supports version '1.0' of the definition schema.",
		}
	}

	return nil
}
