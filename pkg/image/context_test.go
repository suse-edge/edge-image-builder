package image

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext_New(t *testing.T) {
	context, err := NewContext("", "", nil, nil, nil)
	require.NoError(t, err)
	defer os.RemoveAll(context.BuildDir)

	_, err = os.Stat(context.BuildDir)
	require.NoError(t, err)
}

func TestContext_New_ExistingBuildDir(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test
	context, err := NewContext("", tmpDir, nil, nil, nil)
	require.NoError(t, err)

	// Verify
	assert.Contains(t, context.BuildDir, tmpDir)
}
