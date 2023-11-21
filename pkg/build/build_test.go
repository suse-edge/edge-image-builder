package build

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/context"
)

func TestGenerateCombustionDirFilename(t *testing.T) {
	// Setup
	ctx, err := context.NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, context.CleanUpBuildDir(ctx))
	}()

	builder := Builder{
		context: ctx,
	}

	testFilename := "combustion-file.sh"

	// Test
	filename := builder.generateCombustionDirFilename(testFilename)

	// Verify
	expectedFilename := filepath.Join(ctx.CombustionDir, testFilename)
	assert.Equal(t, expectedFilename, filename)
}

func TestGenerateBuildDirFilename(t *testing.T) {
	// Setup
	ctx, err := context.NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, context.CleanUpBuildDir(ctx))
	}()

	builder := Builder{
		context: ctx,
	}

	testFilename := "build-dir-file.sh"

	// Test
	filename := builder.generateBuildDirFilename(testFilename)

	// Verify
	expectedFilename := filepath.Join(ctx.BuildDir, testFilename)
	require.Equal(t, expectedFilename, filename)
}
