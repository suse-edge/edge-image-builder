package combustion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

func TestConfigureCustomFiles(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// - scripts
	fullScriptsDir := filepath.Join(ctx.ImageConfigDir, customDir, customScriptsDir)
	err := os.MkdirAll(fullScriptsDir, os.ModePerm)
	require.NoError(t, err)

	_, err = os.Create(filepath.Join(fullScriptsDir, "foo.sh"))
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(fullScriptsDir, "bar.sh"))
	require.NoError(t, err)

	// - files
	fullFilesDir := filepath.Join(ctx.ImageConfigDir, customDir, customFilesDir)
	err = os.MkdirAll(fullFilesDir, os.ModePerm)
	require.NoError(t, err)

	_, err = os.Create(filepath.Join(fullFilesDir, "baz"))
	require.NoError(t, err)

	// Test
	scripts, err := configureCustomFiles(ctx)

	// Verify
	require.NoError(t, err)

	// - make sure the files were added to the build directory
	foundDirListing, err := os.ReadDir(ctx.CombustionDir)
	require.NoError(t, err)
	assert.Equal(t, 3, len(foundDirListing))

	// - make sure the copied files have the right permissions
	for _, entry := range foundDirListing {
		fullEntryPath := filepath.Join(ctx.CombustionDir, entry.Name())
		stats, err := os.Stat(fullEntryPath)
		require.NoError(t, err)

		if strings.HasSuffix(entry.Name(), ".sh") {
			assert.Equal(t, fileio.ExecutablePerms, stats.Mode())
		} else {
			assert.Equal(t, fileio.NonExecutablePerms, stats.Mode())
		}
	}

	// - make sure only script entries were added to the combustion scripts list
	require.Equal(t, 2, len(scripts))
	assert.Contains(t, scripts, "foo.sh")
	assert.Contains(t, scripts, "bar.sh")
}

func TestConfigureFiles_NoCustomDir(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	scripts, err := configureCustomFiles(ctx)

	// Verify
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestCopyCustomFiles_MissingFromDir(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	files, err := copyCustomFiles("missing", ctx.CombustionDir, fileio.NonExecutablePerms)

	// Verify
	assert.Nil(t, files)
	assert.Nil(t, err)
}

func TestCopyCustomFiles_EmptyFromDir(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// - from directory to look in
	fullScriptsDir := filepath.Join(ctx.ImageConfigDir, customDir, customScriptsDir)
	err := os.MkdirAll(fullScriptsDir, os.ModePerm)
	require.NoError(t, err)

	// Test
	scripts, err := copyCustomFiles(fullScriptsDir, ctx.CombustionDir, fileio.NonExecutablePerms)

	// Verify
	require.Error(t, err)
	assert.ErrorContains(t, err, "no files found in directory")
	assert.Nil(t, scripts)
}
