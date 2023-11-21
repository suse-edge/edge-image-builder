package build

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
