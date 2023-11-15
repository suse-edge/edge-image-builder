package build

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestDeleteNoExistingImage(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	imageConfig := config.ImageConfig{
		Image: config.Image{
			OutputImageName: "not-there",
		},
	}
	context := Context{
		ImageConfigDir: tmpDir,
	}
	builder := New(&imageConfig, &context)

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

	imageConfig := config.ImageConfig{
		Image: config.Image{
			OutputImageName: "not-there",
		},
	}
	context := Context{
		ImageConfigDir: tmpDir,
	}
	builder := New(&imageConfig, &context)

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
	imageConfig := config.ImageConfig{
		Image: config.Image{
			BaseImage:       "base-image",
			OutputImageName: "build-image",
		},
	}
	context := Context{
		ImageConfigDir: "config-dir",
		CombustionDir:  "combustion",
	}
	builder := New(&imageConfig, &context)

	// Test
	cmd, logfile, err := builder.createXorrisoCommand()

	// Verify
	require.NoError(t, err)

	defer os.Remove(builder.generateIsoLogFilename())

	assert.Equal(t, xorrisoExec, cmd.Path)

	expectedString := "/usr/bin/xorriso " +
		"-indev config-dir/images/base-image " +
		"-outdev config-dir/build-image " +
		"-map combustion /combustion " +
		"-boot_image any replay -changes_pending yes"
	expected := strings.Split(expectedString, " ")
	assert.Equal(t, expected, cmd.Args)

	assert.NotNil(t, logfile)
	assert.NotEqual(t, os.Stdout, cmd.Stdout)
	assert.NotEqual(t, os.Stderr, cmd.Stderr)
}
