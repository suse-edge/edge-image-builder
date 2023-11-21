package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCombustionScript(t *testing.T) {
	// Setup
	context, err := NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{
		context: context,
	}

	builder.combustionScripts = append(builder.combustionScripts, "foo.sh", "bar.sh")

	// Test
	err = builder.generateCombustionScript()

	// Verify
	require.NoError(t, err)

	// - check the script contents itself
	scriptBytes, err := os.ReadFile(filepath.Join(context.CombustionDir, "script"))
	require.NoError(t, err)
	scriptData := string(scriptBytes)
	assert.Contains(t, scriptData, "#!/bin/bash")
	assert.Contains(t, scriptData, "foo.sh")
	assert.Contains(t, scriptData, "bar.sh")

	// - ensure the order of the scripts is alphabetical
	assert.Equal(t, "bar.sh", builder.combustionScripts[0])
	assert.Equal(t, "foo.sh", builder.combustionScripts[1])
}

func TestGenerateCombustionDirFilename(t *testing.T) {
	// Setup
	context, err := NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{
		context: context,
	}

	testFilename := "combustion-file.sh"

	// Test
	filename := builder.generateCombustionDirFilename(testFilename)

	// Verify
	expectedFilename := filepath.Join(context.CombustionDir, testFilename)
	assert.Equal(t, expectedFilename, filename)
}

func TestGenerateBuildDirFilename(t *testing.T) {
	// Setup
	context, err := NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{
		context: context,
	}

	testFilename := "build-dir-file.sh"

	// Test
	filename := builder.generateBuildDirFilename(testFilename)

	// Verify
	expectedFilename := filepath.Join(context.BuildDir, testFilename)
	require.Equal(t, expectedFilename, filename)
}
