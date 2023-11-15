package build

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirStructure_New(t *testing.T) {
	ds, err := NewDirStructure("", "", false)
	require.NoError(t, err)
	defer os.RemoveAll(ds.BuildDir)

	_, err = os.Stat(ds.BuildDir)
	require.NoError(t, err)
}

func TestDirStructure_New_ExistingBuildDir(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test
	ds, err := NewDirStructure("", tmpDir, false)
	require.NoError(t, err)

	// Verify
	require.Equal(t, tmpDir, ds.BuildDir)
}

func TestDirStructure_CleanUpBuildDirTrue(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ds := DirStructure{
		BuildDir:       tmpDir,
		DeleteBuildDir: true,
	}

	// Test
	require.NoError(t, ds.CleanUpBuildDir())

	// Verify
	_, err = os.Stat(tmpDir)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestDirStructure_CleanUpBuildDirFalse(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ds := DirStructure{
		BuildDir:       tmpDir,
		DeleteBuildDir: false,
	}

	// Test
	require.NoError(t, ds.CleanUpBuildDir())

	// Verify
	_, err = os.Stat(tmpDir)
	require.NoError(t, err)
}
