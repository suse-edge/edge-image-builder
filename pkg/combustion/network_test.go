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

type mockNetworkConfiguratorInstaller struct {
	installConfiguratorFunc func(imageName, sourcePath, installPath string) error
}

func (m mockNetworkConfiguratorInstaller) InstallConfigurator(imageName, sourcePath, installPath string) error {
	if m.installConfiguratorFunc != nil {
		return m.installConfiguratorFunc(imageName, sourcePath, installPath)
	}

	panic("not implemented")
}

func assertNetworkConfigScript(t *testing.T, scriptPath string) {
	data, err := os.ReadFile(scriptPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "./nmc apply --config-dir network")

	info, err := os.Stat(scriptPath)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, info.Mode())
}

func TestConfigureNetwork_NotConfigured(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	scripts, err := configureNetwork(ctx)
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureNetwork(t *testing.T) {
	tests := []struct {
		name                  string
		configGenerator       mockNetworkConfigGenerator
		configuratorInstaller mockNetworkConfiguratorInstaller
		expectedErr           string
	}{
		{
			name: "Generating config fails",
			configGenerator: mockNetworkConfigGenerator{
				generateNetworkConfigFunc: func(configDir, outputDir string, outputWriter io.Writer) error {
					return fmt.Errorf("no config for you")
				},
			},
			expectedErr: "generating network config: no config for you",
		},
		{
			name: "Installing configurator fails",
			configGenerator: mockNetworkConfigGenerator{
				generateNetworkConfigFunc: func(configDir, outputDir string, outputWriter io.Writer) error {
					return nil
				},
			},
			configuratorInstaller: mockNetworkConfiguratorInstaller{
				installConfiguratorFunc: func(imageName, sourcePath, installPath string) error {
					return fmt.Errorf("no installer for you")
				},
			},
			expectedErr: "installing configurator: no installer for you",
		},
		{
			name: "Successful configuration",
			configGenerator: mockNetworkConfigGenerator{
				generateNetworkConfigFunc: func(configDir, outputDir string, outputWriter io.Writer) error {
					return nil
				},
			},
			configuratorInstaller: mockNetworkConfiguratorInstaller{
				installConfiguratorFunc: func(imageName, sourcePath, installPath string) error {
					return nil
				},
			},
		},
	}

	ctx, teardown := setupContext(t)
	defer teardown()

	networkDir := filepath.Join(ctx.ImageConfigDir, networkConfigDir)
	require.NoError(t, os.Mkdir(networkDir, 0o600))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.NetworkConfigGenerator = test.configGenerator
			ctx.NetworkConfiguratorInstaller = test.configuratorInstaller

			scripts, err := configureNetwork(ctx)

			if test.expectedErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedErr)
				return
			}

			assert.Equal(t, []string{networkConfigScriptName}, scripts)

			scriptPath := filepath.Join(ctx.CombustionDir, networkConfigScriptName)
			assertNetworkConfigScript(t, scriptPath)
		})
	}
}

func TestWriteNetworkConfigurationScript(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	script, err := writeNetworkConfigurationScript(ctx)
	require.NoError(t, err)
	assert.Equal(t, networkConfigScriptName, script)

	scriptPath := filepath.Join(ctx.CombustionDir, script)
	assertNetworkConfigScript(t, scriptPath)
}
