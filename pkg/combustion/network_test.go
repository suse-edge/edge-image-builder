package combustion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

func TestGenerateNetworkConfigCommand(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	var sb strings.Builder

	cmd := generateNetworkConfigCommand(ctx, &sb)

	expectedArgs := []string{
		"nmc",
		"generate",
		"--config-dir", fmt.Sprintf("%s/network", ctx.ImageConfigDir),
		"--output-dir", fmt.Sprintf("%s/network/config", ctx.CombustionDir),
	}

	assert.Equal(t, expectedArgs, cmd.Args)
	assert.Equal(t, &sb, cmd.Stdout)
	assert.Equal(t, &sb, cmd.Stderr)
}

func TestGenerateNetworkConfig_ExecutableMissing(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	err := generateNetworkConfig(ctx)
	require.Error(t, err)
	assert.ErrorContains(t, err, "running generate command")
	assert.ErrorContains(t, err, "executable file not found")
}

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
