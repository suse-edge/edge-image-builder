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
	installConfiguratorFunc func(sourcePath, installPath string) error
}

func (m mockNetworkConfiguratorInstaller) InstallConfigurator(sourcePath, installPath string) error {
	if m.installConfiguratorFunc != nil {
		return m.installConfiguratorFunc(sourcePath, installPath)
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

	var c Combustion

	scripts, err := c.configureNetwork(ctx)
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureNetwork_EmptyDirectory(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	networkDir := filepath.Join(ctx.ImageConfigDir, networkConfigDir)
	require.NoError(t, os.Mkdir(networkDir, 0o700))

	var c Combustion

	scripts, err := c.configureNetwork(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "network directory is present but empty")
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
			name: "Installing configurator fails",
			configuratorInstaller: mockNetworkConfiguratorInstaller{
				installConfiguratorFunc: func(sourcePath, installPath string) error {
					return fmt.Errorf("no installer for you")
				},
			},
			expectedErr: "installing configurator: no installer for you",
		},
		{
			name: "Generating config fails",
			configGenerator: mockNetworkConfigGenerator{
				generateNetworkConfigFunc: func(configDir, outputDir string, outputWriter io.Writer) error {
					return fmt.Errorf("no config for you")
				},
			},
			configuratorInstaller: mockNetworkConfiguratorInstaller{
				installConfiguratorFunc: func(sourcePath, installPath string) error {
					return nil
				},
			},
			expectedErr: "generating network config: no config for you",
		},
		{
			name: "Successful configuration",
			configGenerator: mockNetworkConfigGenerator{
				generateNetworkConfigFunc: func(configDir, outputDir string, outputWriter io.Writer) error {
					return nil
				},
			},
			configuratorInstaller: mockNetworkConfiguratorInstaller{
				installConfiguratorFunc: func(sourcePath, installPath string) error {
					return nil
				},
			},
		},
	}

	ctx, teardown := setupContext(t)
	defer teardown()

	networkDir := filepath.Join(ctx.ImageConfigDir, networkConfigDir)
	require.NoError(t, os.Mkdir(networkDir, 0o700))

	networkConfig := filepath.Join(networkDir, "config.yaml")
	require.NoError(t, os.WriteFile(networkConfig, []byte("some-config"), fileio.NonExecutablePerms))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := Combustion{
				NetworkConfigGenerator:       test.configGenerator,
				NetworkConfiguratorInstaller: test.configuratorInstaller,
			}

			scripts, err := c.configureNetwork(ctx)

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

func TestConfigureNetwork_CustomScript(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	c := Combustion{
		NetworkConfiguratorInstaller: mockNetworkConfiguratorInstaller{
			installConfiguratorFunc: func(sourcePath, installPath string) error {
				return nil
			},
		},
	}

	networkDir := filepath.Join(ctx.ImageConfigDir, networkConfigDir)
	require.NoError(t, os.Mkdir(networkDir, 0o700))

	customScriptPath := filepath.Join(networkDir, networkCustomScriptName)
	customScriptContents := []byte("configure all the nics!")

	require.NoError(t, os.WriteFile(customScriptPath, customScriptContents, 0o600))

	scripts, err := c.configureNetwork(ctx)
	require.NoError(t, err)

	assert.Equal(t, []string{networkConfigScriptName}, scripts)

	scriptPath := filepath.Join(ctx.CombustionDir, networkConfigScriptName)
	contents, err := os.ReadFile(scriptPath)
	require.NoError(t, err)

	assert.Equal(t, customScriptContents, contents)
}

func TestWriteNetworkConfigurationScript(t *testing.T) {
	dir, err := os.MkdirTemp("", "network-config-script-")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	scriptPath := filepath.Join(dir, "script.sh")

	require.NoError(t, writeNetworkConfigurationScript(scriptPath))

	assertNetworkConfigScript(t, scriptPath)
}
