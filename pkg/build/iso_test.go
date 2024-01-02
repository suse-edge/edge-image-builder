package build

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestCreateXorrisoCommand(t *testing.T) {
	// Setup
	builder := Builder{
		context: &image.Context{
			ImageConfigDir: "config-dir",
			CombustionDir:  "combustion",
			ImageDefinition: &image.Definition{
				Image: image.Image{
					BaseImage:       "base-image",
					OutputImageName: "build-image",
				},
			},
		},
	}

	// Test
	cmd, logfile, err := builder.createXorrisoCommand()

	// Verify
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.Remove(logfile.Name()))
	}()

	assert.Equal(t, xorrisoExec, cmd.Path)

	expectedString := "/usr/bin/xorriso " +
		"-indev config-dir/images/base-image " +
		"-outdev config-dir/build-image " +
		"-map combustion /combustion " +
		"-boot_image any replay -changes_pending yes"
	expected := strings.Split(expectedString, " ")
	assert.Equal(t, expected, cmd.Args)

	assert.NotNil(t, logfile)
	assert.Equal(t, logfile, cmd.Stdout)
	assert.Equal(t, logfile, cmd.Stderr)
}
