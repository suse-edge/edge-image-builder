package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRPMFileNames(t *testing.T) {
	// Setup
	dirStructure, err := NewDirStructure("../config/testdata", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, dirStructure.CleanUpBuildDir())
	}()

	builder := &Builder{
		dirStructure: dirStructure,
	}

	rpmSourceDir := filepath.Join(dirStructure.ImageConfigDir, "rpms")

	file1Path := filepath.Join(rpmSourceDir, "rpm1.rpm")
	defer os.Remove(file1Path)
	file1, err := os.Create(file1Path)
	require.NoError(t, err)

	file2Path := filepath.Join(rpmSourceDir, "rpm2.rpm")
	defer os.Remove(file2Path)
	file2, err := os.Create(file2Path)
	require.NoError(t, err)

	// Test
	rpmFileNames, err := builder.getRPMFileNames(rpmSourceDir)

	// Verify
	require.NoError(t, err)

	assert.Contains(t, rpmFileNames, "rpm1.rpm")
	assert.Contains(t, rpmFileNames, "rpm2.rpm")

	// Cleanup
	assert.NoError(t, file1.Close())
	assert.NoError(t, file2.Close())
}

func TestCopyRPMs(t *testing.T) {
	// Setup
	dirStructure, err := NewDirStructure("../config/testdata", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, dirStructure.CleanUpBuildDir())
	}()

	builder := &Builder{
		dirStructure: dirStructure,
	}

	rpmSourceDir := filepath.Join(dirStructure.ImageConfigDir, "rpms")

	file1Path := filepath.Join(rpmSourceDir, "rpm1.rpm")
	defer os.Remove(file1Path)
	file1, err := os.Create(file1Path)
	require.NoError(t, err)

	file2Path := filepath.Join(rpmSourceDir, "rpm2.rpm")
	defer os.Remove(file2Path)
	file2, err := os.Create(file2Path)
	require.NoError(t, err)

	// Test
	err = builder.copyRPMs()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.dirStructure.CombustionDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.dirStructure.CombustionDir, "rpm2.rpm"))
	require.NoError(t, err)

	// Cleanup
	assert.NoError(t, file1.Close())
	assert.NoError(t, file2.Close())
}

func TestGetRPMFileNamesNoRPMs(t *testing.T) {
	// Setup
	dirStructure, err := NewDirStructure("../config/testdata", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, dirStructure.CleanUpBuildDir())
	}()

	builder := &Builder{
		dirStructure: dirStructure,
	}

	rpmSourceDir := filepath.Join(dirStructure.ImageConfigDir, "rpms")

	// Test
	rpmFileNames, err := builder.getRPMFileNames(rpmSourceDir)

	// Verify
	require.ErrorContains(t, err, "no rpms found")

	assert.Empty(t, rpmFileNames)
}

func TestCopyRPMsNoRPMDir(t *testing.T) {
	// Setup
	dirStructure, err := NewDirStructure("../config/ThisDirDoesNotExist", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, dirStructure.CleanUpBuildDir())
	}()

	builder := &Builder{
		dirStructure: dirStructure,
	}

	// Test
	err = builder.copyRPMs()

	// Verify
	require.NoError(t, err)
}
