package build

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestGenerateXorrisoArgs(t *testing.T) {
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
	args := builder.generateXorrisoArgs()

	// Verify
	expected := "-indev config-dir/images/base-image " +
		"-outdev config-dir/build-image " +
		"-map combustion /combustion " +
		"-boot_image any replay -changes_pending yes"
	require.Equal(t, expected, args)
}
