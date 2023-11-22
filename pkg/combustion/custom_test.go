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

func TestConfigureScripts(t *testing.T) {
	// Setup
	// - Testing image config directory
	tmpSrcDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpSrcDir)

	// - scripts directory to look in
	fullScriptsDir := filepath.Join(tmpSrcDir, customScriptsDir)
	err = os.MkdirAll(fullScriptsDir, os.ModePerm)
	require.NoError(t, err)

	// - create sample scripts to be copied
	_, err = os.Create(filepath.Join(fullScriptsDir, "foo.sh"))
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(fullScriptsDir, "bar.sh"))
	require.NoError(t, err)

	// - combustion directory into which the scripts should be copied
	tmpDestDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDestDir)

	ctx := &image.Context{
		ImageConfigDir: tmpSrcDir,
		CombustionDir:  tmpDestDir,
	}

	// Test
	scripts, err := configureCustomScripts(ctx)

	// Verify
	require.NoError(t, err)

	// - make sure the scripts were added to the build directory
	foundDirListing, err := os.ReadDir(tmpDestDir)
	require.NoError(t, err)
	assert.Equal(t, 2, len(foundDirListing))

	// - make sure the copied files have the right permissions
	for _, entry := range foundDirListing {
		fullEntryPath := filepath.Join(ctx.CombustionDir, entry.Name())
		stats, err := os.Stat(fullEntryPath)
		require.NoError(t, err)
		assert.Equal(t, fileio.ExecutablePerms, stats.Mode())
	}

	// - make sure entries were added to the combustion scripts list, so they are
	//   present in the script file that is generated
	assert.Equal(t, 2, len(scripts))
}

func TestConfigureScriptsNoScriptsDir(t *testing.T) {
	// Setup
	tmpSrcDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpSrcDir)

	ctx := &image.Context{
		ImageConfigDir: tmpSrcDir,
	}

	// Test
	scripts, err := configureCustomScripts(ctx)

	// Verify
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureScriptsEmptyScriptsDir(t *testing.T) {
	// Setup
	// - Testing image config directory
	tmpSrcDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpSrcDir)

	// - scripts directory to look in
	fullScriptsDir := filepath.Join(tmpSrcDir, customScriptsDir)
	err = os.MkdirAll(fullScriptsDir, os.ModePerm)
	require.NoError(t, err)

	ctx := &image.Context{
		ImageConfigDir: tmpSrcDir,
	}

	// Test
	scripts, err := configureCustomScripts(ctx)

	// Verify
	require.Error(t, err)
	assert.ErrorContains(t, err, "no scripts found in directory")
	assert.Nil(t, scripts)
}
