package combustion

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func assertNetworkConfigdriveScript(t *testing.T, scriptPath string) {
	data, err := os.ReadFile(scriptPath)
	require.NoError(t, err)

	assert.Contains(t, string(data), "nmc generate --config-dir /tmp/nmc/desired")

	info, err := os.Stat(scriptPath)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, info.Mode())
}

func TestConfigureNetworkConfigdrive_NotConfigured(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	scripts, err := configureNetworkConfigdrive(ctx)
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureNetworkConfigdrive(t *testing.T) {
	tests := []struct {
		name                  string
		configuratorInstaller mockNetworkConfiguratorInstaller
		expectedErr           string
	}{
		{
			name: "Installing configurator fails",
			configuratorInstaller: mockNetworkConfiguratorInstaller{
				installConfiguratorFunc: func(arch image.Arch, sourcePath, installPath string) error {
					return fmt.Errorf("no installer for you")
				},
			},
			expectedErr: "installing configurator: no installer for you",
		},
		{
			name: "Successful configuration",
			configuratorInstaller: mockNetworkConfiguratorInstaller{
				installConfiguratorFunc: func(arch image.Arch, sourcePath, installPath string) error {
					return nil
				},
			},
		},
	}

	ctx, teardown := setupContext(t)
	defer teardown()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.NetworkConfiguratorInstaller = test.configuratorInstaller
			ctx.ImageDefinition.OperatingSystem.ConfigDrive = true

			scripts, err := configureNetworkConfigdrive(ctx)

			if test.expectedErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedErr)
				return
			}

			assert.Equal(t, []string{networkConfigdriveScriptName}, scripts)

			scriptPath := filepath.Join(ctx.CombustionDir, networkConfigdriveScriptName)
			assertNetworkConfigdriveScript(t, scriptPath)
		})
	}
}

func TestWriteNetworkConfigdriveScript(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	script, err := writeNetworkConfigdriveScript(ctx)
	require.NoError(t, err)
	assert.Equal(t, networkConfigdriveScriptName, script)

	scriptPath := filepath.Join(ctx.CombustionDir, script)
	assertNetworkConfigdriveScript(t, scriptPath)
}
