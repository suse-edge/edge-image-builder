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
	builder := New(nil, &bc)

	// Test
	err := builder.prepareBuildDir()
	defer os.Remove(bc.BuildTempDir)

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(bc.BuildTempDir)
	require.NoError(t, err)
	_, err = os.Stat(builder.combustionDir)
	require.NoError(t, err)
}

func TestCleanUpBuildDirWithDelete(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		DeleteArtifacts: true,
	}
	builder := New(nil, &bc)

	testDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	bc.BuildTempDir = testDir

	// Test
	err = builder.cleanUpBuildDir()

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
	builder := New(nil, &bc)
	testDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	bc.BuildTempDir = testDir

	// Test
	err = builder.cleanUpBuildDir()
	defer os.Remove(bc.BuildTempDir)

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(bc.BuildTempDir)
	require.NoError(t, err)
}

func TestGenerateCombustionScript(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(bc.BuildTempDir)

	builder.combustionScripts = append(builder.combustionScripts, "foo.sh", "bar.sh")

	// Test
	err = builder.generateCombustionScript()

	// Verify
	require.NoError(t, err)

	scriptBytes, err := os.ReadFile(filepath.Join(builder.combustionDir, "script"))
	scriptData := string(scriptBytes)
	assert.Contains(t, scriptData, "#!/bin/bash")
	assert.Contains(t, scriptData, "foo.sh")
	assert.Contains(t, scriptData, "bar.sh")
}