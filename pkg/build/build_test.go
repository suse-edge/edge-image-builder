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
	defer os.Remove(builder.eibBuildDir)

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(builder.eibBuildDir)
	require.NoError(t, err)
}

func TestPrepareBuildDirExistingDir(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.Remove(tmpDir)

	bc := config.BuildConfig{BuildDir: tmpDir}
	builder := New(nil, &bc)

	// Test
	err = builder.prepareBuildDir()

	// Verify
	require.NoError(t, err)
	require.Equal(t, tmpDir, builder.eibBuildDir)
}

func TestGenerateCombustionScript(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	builder.combustionScripts = append(builder.combustionScripts, "foo.sh", "bar.sh")

	// Test
	err = builder.generateCombustionScript()

	// Verify
	require.NoError(t, err)

	scriptBytes, err := os.ReadFile(filepath.Join(builder.combustionDir, "script"))
	require.NoError(t, err)
	scriptData := string(scriptBytes)
	assert.Contains(t, scriptData, "#!/bin/bash")
	assert.Contains(t, scriptData, "foo.sh")
	assert.Contains(t, scriptData, "bar.sh")
}
