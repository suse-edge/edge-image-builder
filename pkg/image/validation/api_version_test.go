package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestValidateApiVersion(t *testing.T) {
	tests := map[string]struct {
		Definition            image.Definition
		ExpectedFailedMessage string
	}{
		`valid`: {
			Definition: image.Definition{
				APIVersion: "1.0",
			},
		},
		`invalid`: {
			Definition: image.Definition{
				APIVersion: "1.1",
			},
			ExpectedFailedMessage: "This version of Edge Image Builder only supports version '1.0' of the definition schema.",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			def := test.Definition
			ctx := image.Context{
				ImageDefinition: &def,
			}
			failure := validateAPIVersion(&ctx)
			if test.ExpectedFailedMessage == "" {
				assert.Nil(t, failure)
			} else {
				assert.Equal(t, test.ExpectedFailedMessage, failure.UserMessage)
			}
		})
	}
}
