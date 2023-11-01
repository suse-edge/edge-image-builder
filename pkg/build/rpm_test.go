package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestGetRPMFileNames(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		ImageConfigDir: "../config/testdata",
	}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	rpmSourceDir := filepath.Join(builder.buildConfig.ImageConfigDir, "rpms")

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
	bc := config.BuildConfig{
		ImageConfigDir: "../config/testdata",
	}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	rpmSourceDir := filepath.Join(builder.buildConfig.ImageConfigDir, "rpms")
	rpmDestDir := builder.combustionDir

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

	_, err = os.Stat(filepath.Join(rpmDestDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(rpmDestDir, "rpm2.rpm"))
	require.NoError(t, err)

	// Cleanup
	assert.NoError(t, file1.Close())
	assert.NoError(t, file2.Close())
}

func TestGetRPMFileNamesNoRPMs(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		ImageConfigDir: "../config/testdata",
	}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	rpmSourceDir := filepath.Join(builder.buildConfig.ImageConfigDir, "rpms")

	// Test
	rpmFileNames, err := builder.getRPMFileNames(rpmSourceDir)

	// Verify
	require.ErrorContains(t, err, "no rpms found")

	assert.Empty(t, rpmFileNames)
}
