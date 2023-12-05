package image

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext_New(t *testing.T) {
	context, err := NewContext("", "", false, nil, nil, nil)
	require.NoError(t, err)
	defer os.RemoveAll(context.BuildDir)

	_, err = os.Stat(context.BuildDir)
	require.NoError(t, err)
}

func TestContext_New_ExistingBuildDir(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test
	context, err := NewContext("", tmpDir, false, nil, nil, nil)
	require.NoError(t, err)

	// Verify
	require.Equal(t, tmpDir, context.BuildDir)
}

func TestCleanUpBuildDirTrue(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	context := &Context{
		BuildDir:       tmpDir,
		DeleteBuildDir: true,
	}

	// Test
	require.NoError(t, CleanUpBuildDir(context))

	// Verify
	_, err = os.Stat(tmpDir)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestCleanUpBuildDirFalse(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	context := &Context{
		BuildDir:       tmpDir,
		DeleteBuildDir: false,
	}

	// Test
	require.NoError(t, CleanUpBuildDir(context))

	// Verify
	_, err = os.Stat(tmpDir)
	require.NoError(t, err)
}
