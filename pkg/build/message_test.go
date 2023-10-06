package build

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestConfigureMessage(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	err := prepareBuildDir(&bc)
	require.NoError(t, err)
	defer os.Remove(bc.BuildTempDir)

	// Test
	err = ConfigureMessage(&bc)

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(bc.CombustionDir, messageScriptName))
	require.NoError(t, err)

	require.Equal(t, 1, len(bc.CombustionScripts))
	assert.Equal(t, messageScriptName, bc.CombustionScripts[0])
}
