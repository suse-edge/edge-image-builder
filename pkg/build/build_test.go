package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestPrepareBuildDir(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}

	// Test
	err := prepareBuildDir(&bc)
	defer os.Remove(bc.BuildTempDir)

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(bc.BuildTempDir)
	require.NoError(t, err)
	_, err = os.Stat(bc.CombustionDir)
	require.NoError(t, err)
}

func TestCleanUpBuildDirWithDelete(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		DeleteArtifacts: true,
	}
	testDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	bc.BuildTempDir = testDir

	// Test
	err = cleanUpBuildDir(&bc)

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(bc.BuildTempDir)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestCleanUpBuildDirNoDelete(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		DeleteArtifacts: false,
	}
	testDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	bc.BuildTempDir = testDir

	// Test
	err = cleanUpBuildDir(&bc)
	defer os.Remove(bc.BuildTempDir)

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(bc.BuildTempDir)
	require.NoError(t, err)
}

func TestGenerateCombustionScript(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	err := prepareBuildDir(&bc)
	require.NoError(t, err)
	defer os.Remove(bc.BuildTempDir)

	bc.AddCombustionScript("foo.sh")
	bc.AddCombustionScript("bar.sh")

	// Test
	err = generateCombustionScript(&bc)

	// Verify
	require.NoError(t, err)

	scriptBytes, err := os.ReadFile(filepath.Join(bc.CombustionDir, "script"))
	scriptData := string(scriptBytes)
	assert.Contains(t, scriptData, "#!/bin/bash")
	assert.Contains(t, scriptData, "foo.sh")
	assert.Contains(t, scriptData, "bar.sh")
}