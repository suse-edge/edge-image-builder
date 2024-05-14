package combustion

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

func TestConfigureCustomFiles(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	scriptsDir := filepath.Join(ctx.ImageConfigDir, customDir, customScriptsDir)
	require.NoError(t, os.MkdirAll(scriptsDir, os.ModePerm))

	filesDir := filepath.Join(ctx.ImageConfigDir, customDir, customFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, os.ModePerm))

	files := map[string]struct {
		isScript bool
		perms    fs.FileMode
	}{
		"foo.sh": {
			isScript: true,
			perms:    0o744,
		},
		"bar.sh": {
			isScript: true,
			perms:    0o755,
		},
		"baz": {
			isScript: false,
			perms:    0o744,
		},
		"qux": {
			isScript: false,
			perms:    0o644,
		},
	}

	for filename, info := range files {
		var path string

		if info.isScript {
			path = filepath.Join(scriptsDir, filename)
		} else {
			path = filepath.Join(filesDir, filename)
		}

		require.NoError(t, os.WriteFile(path, nil, info.perms))
	}

	// Test
	scripts, err := configureCustomFiles(ctx)

	// Verify
	require.NoError(t, err)

	// - make sure the files were added to the build directory
	dirEntries, err := os.ReadDir(ctx.CombustionDir)
	require.NoError(t, err)
	require.Len(t, dirEntries, 4)

	// - make sure the copied files have the right permissions
	for _, entry := range dirEntries {
		file, ok := files[entry.Name()]
		require.Truef(t, ok, "Unexpected file: %s", entry.Name())

		entryPath := filepath.Join(ctx.CombustionDir, entry.Name())
		stats, err := os.Stat(entryPath)
		require.NoError(t, err)

		if files[entry.Name()].isScript {
			assert.Equal(t, fileio.ExecutablePerms, stats.Mode())
		} else {
			assert.Equal(t, file.perms, stats.Mode())
		}
	}

	// - make sure only script entries were added to the combustion scripts list
	require.Len(t, scripts, 2)
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
	files, err := copyCustomFiles("missing", ctx.CombustionDir, nil)

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
	scripts, err := copyCustomFiles(fullScriptsDir, ctx.CombustionDir, nil)

	// Verify
	require.Error(t, err)
	assert.ErrorContains(t, err, "no files found in directory")
	assert.Nil(t, scripts)
}
