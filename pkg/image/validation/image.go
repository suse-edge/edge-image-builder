package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	imageComponent = "image"
)

var validImageTypes = []string{image.TypeISO, image.TypeRAW}
var validArchTypes = []string{string(image.ArchTypeARM), string(image.ArchTypeX86)}

func validateImage(ctx *image.Context) []FailedValidation {
	def := ctx.ImageDefinition

	var failures []FailedValidation

	if def.Image.ImageType == "" {
		failures = append(failures, FailedValidation{
			userMessage: "The 'imageType' field is required in the 'image' section.",
			component:   imageComponent,
		})
	} else if !slices.Contains(validImageTypes, def.Image.ImageType) {
		msg := fmt.Sprintf("The 'imageType' field must be one of: %s", strings.Join(validImageTypes, ", "))
		failures = append(failures, FailedValidation{
			userMessage: msg,
			component:   imageComponent,
		})
	}

	if def.Image.Arch == "" {
		failures = append(failures, FailedValidation{
			userMessage: "The 'arch' field is required in the 'image' section.",
			component:   imageComponent,
		})
	} else if !slices.Contains(validArchTypes, string(def.Image.Arch)) {
		msg := fmt.Sprintf("The 'arch' field must be one of: %s", strings.Join(validArchTypes, ", "))
		failures = append(failures, FailedValidation{
			userMessage: msg,
			component:   imageComponent,
		})
	}

	if def.Image.OutputImageName == "" {
		failures = append(failures, FailedValidation{
			userMessage: "The 'outputImageName' field is required in the 'image' section.",
			component:   imageComponent,
		})
	}

	if def.Image.BaseImage == "" {
		failures = append(failures, FailedValidation{
			userMessage: "The 'baseImage' field is required in the 'image' section.",
			component:   imageComponent,
		})
	} else {
		baseImageFilename := filepath.Join(ctx.ImageConfigDir, "images", def.Image.BaseImage)
		_, err := os.Stat(baseImageFilename)
		if err != nil {
			if os.IsNotExist(err) {
				msg := fmt.Sprintf("The specified base image '%s' cannot be found.", def.Image.BaseImage)
				failures = append(failures, FailedValidation{
					userMessage: msg,
					component:   imageComponent,
				})
			} else {
				msg := fmt.Sprintf("The specified base image '%s' cannot be read. See the logs for more information.", def.Image.BaseImage)
				failures = append(failures, FailedValidation{
					userMessage: msg,
					component:   imageComponent,
					err:         err,
				})
			}
		}
	}

	return failures
}
