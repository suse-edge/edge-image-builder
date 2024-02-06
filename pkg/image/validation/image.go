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
	imageComponent = "Image"
)

func validateImage(ctx *image.Context) []FailedValidation {
	def := ctx.ImageDefinition

	validImageTypes := []string{image.TypeISO, image.TypeRAW}
	validArchTypes := []string{string(image.ArchTypeARM), string(image.ArchTypeX86)}

	var failures []FailedValidation

	if def.Image.ImageType == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'imageType' field is required in the 'image' section.",
		})
	} else if !slices.Contains(validImageTypes, def.Image.ImageType) {
		msg := fmt.Sprintf("The 'imageType' field must be one of: %s", strings.Join(validImageTypes, ", "))
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	if def.Image.Arch == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'arch' field is required in the 'image' section.",
		})
	} else if !slices.Contains(validArchTypes, string(def.Image.Arch)) {
		msg := fmt.Sprintf("The 'arch' field must be one of: %s", strings.Join(validArchTypes, ", "))
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	if def.Image.OutputImageName == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'outputImageName' field is required in the 'image' section.",
		})
	}

	if def.Image.BaseImage == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'baseImage' field is required in the 'image' section.",
		})
	} else {
		baseImageFilename := filepath.Join(ctx.ImageConfigDir, "base-images", def.Image.BaseImage)
		_, err := os.Stat(baseImageFilename)
		if err != nil {
			if os.IsNotExist(err) {
				msg := fmt.Sprintf("The specified base image '%s' cannot be found.", def.Image.BaseImage)
				failures = append(failures, FailedValidation{
					UserMessage: msg,
				})
			} else {
				msg := fmt.Sprintf("The specified base image '%s' cannot be read. See the logs for more information.", def.Image.BaseImage)
				failures = append(failures, FailedValidation{
					UserMessage: msg,
					Error:       err,
				})
			}
		}
	}

	return failures
}
