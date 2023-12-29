package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestSetupBuildDirectory(t *testing.T) {
	buildDir, combustionDir, err := SetupBuildDirectory("")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(buildDir))
	}()

	_, err = os.Stat(buildDir)
	require.NoError(t, err)

	_, err = os.Stat(combustionDir)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(buildDir, "combustion"), combustionDir)
}

func TestSetupBuildDirectory_ExistingRootDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	buildDir, combustionDir, err := SetupBuildDirectory(tmpDir)
	require.NoError(t, err)

	_, err = os.Stat(buildDir)
	require.NoError(t, err)

	_, err = os.Stat(combustionDir)
	require.NoError(t, err)

	assert.Contains(t, buildDir, filepath.Join(tmpDir, "build-"))
	assert.Equal(t, filepath.Join(buildDir, "combustion"), combustionDir)
}

func TestGenerateBuildDirFilename(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	builder := Builder{
		context: &image.Context{
			BuildDir: tmpDir,
		},
	}

	testFilename := "build-dir-file.sh"

	// Test
	filename := builder.generateBuildDirFilename(testFilename)

	// Verify
	expectedFilename := filepath.Join(builder.context.BuildDir, testFilename)
	require.Equal(t, expectedFilename, filename)
}
