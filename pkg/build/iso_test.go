package build

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestDeleteNoExistingImage(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	builder := Builder{
		context: &image.Context{
			ImageConfigDir: tmpDir,
			ImageDefinition: &image.Definition{
				Image: image.Image{
					OutputImageName: "not-there",
				},
			},
		},
	}

	// Test
	err = builder.deleteExistingOutputIso()

	// Verify
	require.NoError(t, err)
}

func TestDeleteExistingImage(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	builder := Builder{
		context: &image.Context{
			ImageConfigDir: tmpDir,
			ImageDefinition: &image.Definition{
				Image: image.Image{
					OutputImageName: "not-there",
				},
			},
		},
	}

	_, err = os.Create(builder.generateOutputImageFilename())
	require.NoError(t, err)

	// Test
	err = builder.deleteExistingOutputIso()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(builder.generateOutputImageFilename())
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}
