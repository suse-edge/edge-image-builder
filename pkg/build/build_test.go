package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestGenerateBuildDirFilename(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	builder := Builder{
		context: &image.Context{
			BuildDir: tmpDir,
		},
	}

	testFilename := "build-dir-file.sh"

	// Test
	filename := builder.generateBuildDirFilename(testFilename)

	// Verify
	expectedFilename := filepath.Join(builder.context.BuildDir, testFilename)
	require.Equal(t, expectedFilename, filename)
}

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
	err = builder.deleteExistingOutputImage()

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
	err = builder.deleteExistingOutputImage()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(builder.generateOutputImageFilename())
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}
