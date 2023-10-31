package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestCreateRawImageCopyCommand(t *testing.T) {
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

	// Test
	cmd := builder.createRawImageCopyCommand()

	// Verify
	require.NotNil(t, cmd)

	assert.Equal(t, copyExec, cmd.Path)
	expectedArgs := []string{
		copyExec,
		builder.generateBaseImageFilename(),
		builder.generateOutputImageFilename(),
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

func TestWriteModifyScript(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	imageConfig := config.ImageConfig{
		Image: config.Image{
			OutputImageName: "output-image",
		},
	}
	buildConfig := config.BuildConfig{
		ImageConfigDir: "config-dir",
		BuildDir:       tmpDir,
	}
	builder := New(&imageConfig, &buildConfig)
	require.NoError(t, builder.prepareBuildDir())

	// Test
	err = builder.writeModifyScript()

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(tmpDir, modifyScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "guestfish --rw -a config-dir/output-image")
	assert.Contains(t, foundContents, "copy-in "+builder.combustionDir)
}

func TestCreateModifyCommand(t *testing.T) {
	// Setup
	buildConfig := config.BuildConfig{
		BuildDir: "build-dir",
	}
	builder := New(nil, &buildConfig)

	// Test
	cmd := builder.createModifyCommand()

	// Verify
	require.NotNil(t, cmd)

	expectedPath := filepath.Join("build-dir", modifyScriptName)
	assert.Equal(t, expectedPath, cmd.Path)
}
