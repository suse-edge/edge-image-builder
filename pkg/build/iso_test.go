package build

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestCreateXorrisoCommand(t *testing.T) {
	// Setup
	imageConfig := config.ImageConfig{
		Image: config.Image{
			BaseImage:       "base-image",
			OutputImageName: "build-image",
		},
	}
	buildConfig := config.BuildConfig{
		ImageConfigDir: "config-dir",
	}
	builder := New(&imageConfig, &buildConfig)
	builder.combustionDir = "combustion"

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
