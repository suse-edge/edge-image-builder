package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

func TestWriteNetworkConfigurationScript(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	script, err := writeNetworkConfigurationScript(ctx)
	require.NoError(t, err)
	assert.Equal(t, networkConfigScriptName, script)

	scriptPath := filepath.Join(ctx.CombustionDir, script)
	data, err := os.ReadFile(scriptPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "./network/nmc apply --config-dir network/config")

	info, err := os.Stat(scriptPath)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, info.Mode())
}
