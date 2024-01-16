package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestConfigureKeymap(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Keymap: "gb",
		},
	}

	// Test
	scripts, err := configureKeymap(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, keymapScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, keymapScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - Make sure that the keymap is set correctly
	assert.Contains(t, foundContents, "echo \"KEYMAP=gb\" >> /etc/vconsole.conf", "keymap not correctly set")
}

func TestConfigureKeymap_NoConf(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{},
	}

	// Test
	scripts, err := configureKeymap(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, keymapScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, keymapScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - Make sure that the keymap is set correctly
	assert.Contains(t, foundContents, "echo \"KEYMAP=us\" >> /etc/vconsole.conf", "keymap not correctly set")
}
