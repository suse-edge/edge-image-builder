package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestValidateImage(t *testing.T) {
	imageConfigDir, err := os.MkdirTemp("", "eib-image-tests-")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(imageConfigDir)
	}()

	testImagesDir := filepath.Join(imageConfigDir, "base-images")
	err = os.Mkdir(testImagesDir, os.ModePerm)
	require.NoError(t, err)

	testBaseImageFilename := filepath.Join(testImagesDir, "base-image.iso")
	_, err = os.Create(testBaseImageFilename)
	require.NoError(t, err)

	tests := map[string]struct {
		ImageDefinition        image.Definition
		ExpectedFailedMessages []string
	}{
		`complete valid definition`: {
			ImageDefinition: image.Definition{
				Image: image.Image{
					ImageType:       image.TypeISO,
					Arch:            image.ArchTypeX86,
					BaseImage:       "base-image.iso",
					OutputImageName: "eib-created.iso",
				},
			},
		},
		`missing all fields`: {
			ImageDefinition: image.Definition{
				Image: image.Image{},
			},
			ExpectedFailedMessages: []string{
				"The 'imageType' field is required in the 'image' section.",
				"The 'arch' field is required in the 'image' section.",
				"The 'outputImageName' field is required in the 'image' section.",
				"The 'baseImage' field is required in the 'image' section.",
			},
		},
		`invalid enum values`: {
			ImageDefinition: image.Definition{
				Image: image.Image{
					ImageType:       "foo",
					Arch:            "bar",
					BaseImage:       "base-image.iso",
					OutputImageName: "eib-created.iso",
				},
			},
			ExpectedFailedMessages: []string{
				"The 'imageType' field must be one of: iso, raw",
				"The 'arch' field must be one of: aarch64, x86_64",
			},
		},
		`base image not found`: {
			ImageDefinition: image.Definition{
				Image: image.Image{
					ImageType:       image.TypeISO,
					Arch:            image.ArchTypeX86,
					BaseImage:       "not-there",
					OutputImageName: "eib-created.iso",
				},
			},
			ExpectedFailedMessages: []string{
				"The specified base image 'not-there' cannot be found.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			imageDef := test.ImageDefinition
			ctx := image.Context{
				ImageConfigDir:  imageConfigDir,
				ImageDefinition: &imageDef,
			}
			failedValidations := validateImage(&ctx)
			assert.Len(t, failedValidations, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failedValidations {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}
