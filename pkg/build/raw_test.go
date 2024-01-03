package build

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestCreateRawImageCopyCommand(t *testing.T) {
	// Setup
	builder := Builder{
		context: &image.Context{
			ImageConfigDir: "config-dir",
			ImageDefinition: &image.Definition{
				Image: image.Image{
					BaseImage:       "base-image",
					OutputImageName: "build-image",
				},
			},
		},
	}

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

	builder := Builder{
		context: &image.Context{
			ImageConfigDir: "config-dir",
			BuildDir:       tmpDir,
			ImageDefinition: &image.Definition{
				Image: image.Image{
					OutputImageName: "output-image",
				},
				OperatingSystem: image.OperatingSystem{
					KernelArgs: []string{"alpha", "beta"},
				},
			},
		},
	}

	// Test
	err = builder.writeModifyScript()

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(tmpDir, modifyScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "guestfish --format=raw --rw -a config-dir/output-image")
	assert.Contains(t, foundContents, "copy-in "+builder.context.CombustionDir)
	assert.Contains(t, foundContents, "download /boot/grub2/grub.cfg /tmp/grub.cfg")
}

func TestCreateModifyCommand(t *testing.T) {
	// Setup
	builder := Builder{
		context: &image.Context{
			BuildDir: "build-dir",
		},
	}

	// Test
	cmd := builder.createModifyCommand(io.Discard)

	// Verify
	require.NotNil(t, cmd)

	expectedPath := filepath.Join("build-dir", modifyScriptName)
	assert.Equal(t, expectedPath, cmd.Path)
	assert.Equal(t, io.Discard, cmd.Stdout)
	assert.Equal(t, io.Discard, cmd.Stderr)
}
