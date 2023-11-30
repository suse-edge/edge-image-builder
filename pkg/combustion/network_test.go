package combustion

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

type mockNetworkConfigGenerator struct {
	generateNetworkConfigFunc func(configDir, outputDir string, outputWriter io.Writer) error
}

func (m mockNetworkConfigGenerator) GenerateNetworkConfig(configDir, outputDir string, outputWriter io.Writer) error {
	if m.generateNetworkConfigFunc != nil {
		return m.generateNetworkConfigFunc(configDir, outputDir, outputWriter)
	}

	panic("not implemented")
}

func TestConfigureNetwork_NotConfigured(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	scripts, err := configureNetwork(ctx)
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureNetwork_GenerateConfigError(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	networkDir := filepath.Join(ctx.ImageConfigDir, networkConfigDir)
	require.NoError(t, os.Mkdir(networkDir, 0o600))

	ctx.NetworkConfigGenerator = mockNetworkConfigGenerator{
		generateNetworkConfigFunc: func(configDir, outputDir string, outputWriter io.Writer) error {
			return fmt.Errorf("no config for you")
		},
	}

	scripts, err := configureNetwork(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "generating network config: no config for you")

	assert.Nil(t, scripts)
}

func TestConfigureNetwork_CopyExecutableError(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	networkDir := filepath.Join(ctx.ImageConfigDir, networkConfigDir)
	require.NoError(t, os.Mkdir(networkDir, 0o600))

	ctx.NetworkConfigGenerator = mockNetworkConfigGenerator{
		generateNetworkConfigFunc: func(configDir, outputDir string, outputWriter io.Writer) error {
			return nil
		},
	}

	scripts, err := configureNetwork(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "writing nmc executable: searching for executable: exec: \"nmc\": executable file not found in $PATH")

	assert.Nil(t, scripts)
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

	assert.Contains(t, string(data), "./nmc apply --config-dir network")

	info, err := os.Stat(scriptPath)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, info.Mode())
}
