package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestConfigureScripts(t *testing.T) {
	// Setup
	// - Testing image config directory
	tmpSrcDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpSrcDir)

	// - scripts directory to look in
	fullScriptsDir := filepath.Join(tmpSrcDir, scriptsDir)
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

	builder := New(nil, &config.BuildConfig{ImageConfigDir: tmpSrcDir})
	builder.combustionDir = tmpDestDir

	// Test
	err = builder.configureScripts()

	// Verify
	require.NoError(t, err)

	// - make sure the scripts were added to the build directory
	foundDirListing, err := os.ReadDir(tmpDestDir)
	require.NoError(t, err)
	assert.Equal(t, 2, len(foundDirListing))

	// - make sure the copied files have the right permissions
	for _, entry := range foundDirListing {
		fullEntryPath := filepath.Join(builder.combustionDir, entry.Name())
		stats, err := os.Stat(fullEntryPath)
		require.NoError(t, err)
		foundMode := stats.Mode()
		assert.Equal(t, "-rwxr--r--", foundMode.String())
	}

	// - make sure entries were added to the combustion scripts list, so they are
	//   present in the script file that is generated
	assert.Equal(t, 2, len(builder.combustionScripts))
}

func TestConfigureScriptsNoScriptsDir(t *testing.T) {
	// Setup
	tmpSrcDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpSrcDir)

	builder := New(nil, &config.BuildConfig{ImageConfigDir: tmpSrcDir})

	// Test
	err = builder.configureScripts()

	// Verify
	require.NoError(t, err)
}

func TestConfigureScriptsEmptyScriptsDir(t *testing.T) {
	// Setup
	// - Testing image config directory
	tmpSrcDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpSrcDir)

	// - scripts directory to look in
	fullScriptsDir := filepath.Join(tmpSrcDir, scriptsDir)
	err = os.MkdirAll(fullScriptsDir, os.ModePerm)
	require.NoError(t, err)

	builder := New(nil, &config.BuildConfig{ImageConfigDir: tmpSrcDir})

	// Test
	err = builder.configureScripts()

	// Verify
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no scripts found in directory")
}
