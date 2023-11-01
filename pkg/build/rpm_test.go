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

	file1Path := filepath.Join(builder.buildConfig.ImageConfigDir, "rpms", "rpm1.rpm")
	file2Path := filepath.Join(builder.buildConfig.ImageConfigDir, "rpms", "rpm2.rpm")

	file1, err := os.Create(file1Path)
	require.NoError(t, err)

	file2, err := os.Create(file2Path)
	require.NoError(t, err)

	// Test
	rpmFileNames, err := builder.getRPMFileNames()

	// Verify
	require.NoError(t, err)

	assert.Contains(t, rpmFileNames, "rpm1.rpm")
	assert.Contains(t, rpmFileNames, "rpm2.rpm")

	// Cleanup
	err = file1.Close()
	require.NoError(t, err)

	err = file2.Close()
	require.NoError(t, err)

	err = os.Remove(file1Path)
	require.NoError(t, err)

	err = os.Remove(file2Path)
	require.NoError(t, err)
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

	file1Path := filepath.Join(builder.buildConfig.ImageConfigDir, "rpms", "rpm1.rpm")
	file2Path := filepath.Join(builder.buildConfig.ImageConfigDir, "rpms", "rpm2.rpm")

	file1, err := os.Create(file1Path)
	require.NoError(t, err)

	file2, err := os.Create(file2Path)
	require.NoError(t, err)

	// Test
	err = builder.copyRPMs()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, "rpm2.rpm"))
	require.NoError(t, err)

	// Cleanup
	err = file1.Close()
	require.NoError(t, err)

	err = file2.Close()
	require.NoError(t, err)

	err = os.Remove(file1Path)
	require.NoError(t, err)

	err = os.Remove(file2Path)
	require.NoError(t, err)
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

	// Test
	rpmFileNames, err := builder.getRPMFileNames()

	// Verify
	require.ErrorContains(t, err, "no rpms found")

	assert.Empty(t, rpmFileNames)
}
