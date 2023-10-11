package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestConfigureMessage(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	// Test
	err = builder.configureMessage()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, messageScriptName))
	require.NoError(t, err)

	require.Equal(t, 1, len(builder.combustionScripts))
	assert.Equal(t, messageScriptName, builder.combustionScripts[0])
}
