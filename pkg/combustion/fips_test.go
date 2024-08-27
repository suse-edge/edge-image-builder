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

func TestConfigureFips_NoConf(t *testing.T) {
	// Setup
	var ctx image.Context

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{},
	}

	// Test
	scripts, err := configureFips(&ctx)

	// Verify
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureFips_Enabled(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			EnableFips: true,
		},
	}

	// Test
	scripts, err := configureFips(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, fipsScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, fipsScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - Ensure that we have the fips setup script defined
	assert.Contains(t, foundContents, "fips-mode-setup --enable", "fips setup script missing")
}
