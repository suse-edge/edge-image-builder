package network

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfiguratorInstaller_InstallConfigurator_InvalidArch(t *testing.T) {
	var installer ConfiguratorInstaller

	err := installer.InstallConfigurator("abc", "def")
	require.Error(t, err)
	assert.EqualError(t, err, "failed to determine arch of image abc")
}

func TestConfiguratorInstaller_InstallConfigurator_AMD(t *testing.T) {
	t.Skip()

	tmpDir, err := os.MkdirTemp("", "eib-configurator-installer-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	var installer ConfiguratorInstaller

	installPath := filepath.Join(tmpDir, "nmc")
	require.NoError(t, installer.InstallConfigurator("abc-x86_64", installPath))
}

func TestConfiguratorInstaller_InstallConfigurator_ARM(t *testing.T) {
	t.Skip()

	tmpDir, err := os.MkdirTemp("", "eib-configurator-installer-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	var installer ConfiguratorInstaller

	installPath := filepath.Join(tmpDir, "nmc")
	require.NoError(t, installer.InstallConfigurator("abc-aarch64", installPath))
}
