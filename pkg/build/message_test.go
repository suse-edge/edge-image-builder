package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureMessage(t *testing.T) {
	// Setup
	dirStructure, err := NewDirStructure("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, dirStructure.CleanUpBuildDir())
	}()

	builder := Builder{dirStructure: dirStructure}

	// Test
	err = builder.configureMessage()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.dirStructure.CombustionDir, messageScriptName))
	require.NoError(t, err)

	require.Equal(t, 1, len(builder.combustionScripts))
	assert.Equal(t, messageScriptName, builder.combustionScripts[0])
}
