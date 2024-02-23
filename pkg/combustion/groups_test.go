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

func TestConfigureGroups(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Groups: []image.OperatingSystemGroup{
				{
					Name: "group1",
					GID:  1000,
				},
				{
					Name: "group2",
				},
			},
		},
	}

	// Test
	scripts, err := configureGroups(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, groupsScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, groupsScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	assert.Contains(t, foundContents, "groupadd -f -g 1000 group1")
	assert.Contains(t, foundContents, "groupadd -f group2")
}
