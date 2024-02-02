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

func TestConfigureSuma_NoConf(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Suma: image.Suma{},
		},
	}

	// Test
	scripts, err := configureSuma(ctx)

	// Verify
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureSuma_FullConfiguration(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Suma: image.Suma{
				Host:          "suma.edge.suse.com",
				ActivationKey: "slemicro55",
			},
		},
	}

	// Test
	scripts, err := configureSuma(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, sumaScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, sumaScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - Ensure that we have the correct URL defined
	assert.Contains(t, foundContents, "master: suma.edge.suse.com")

	// - Ensure that we've got the activation key defined
	assert.Contains(t, foundContents, "activation_key: \"slemicro55\"")
}
