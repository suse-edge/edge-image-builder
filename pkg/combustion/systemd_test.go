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

func TestConfigureSystemd_NoServices(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Systemd: image.Systemd{},
		},
	}

	// Test
	scripts, err := configureSystemd(ctx)

	// Verify
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureSystemd_BothServiceTypes(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Systemd: image.Systemd{
				Enable:  []string{"enable0"},
				Disable: []string{"disable0", "disable1"},
			},
		},
	}

	// Test
	scripts, err := configureSystemd(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, systemdScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, systemdScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - Enabled services
	assert.Contains(t, foundContents, "systemctl enable enable0")

	// - Disabled services
	assert.Contains(t, foundContents, "systemctl disable disable0")
	assert.Contains(t, foundContents, "systemctl mask disable0")
	assert.Contains(t, foundContents, "systemctl disable disable1")
	assert.Contains(t, foundContents, "systemctl mask disable1")
}
