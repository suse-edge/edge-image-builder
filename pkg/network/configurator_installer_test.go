package network

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

func TestConfiguratorInstaller_InstallConfigurator(t *testing.T) {
	binaryContents := []byte("network magic")

	srcDir, err := os.MkdirTemp("", "eib-configurator-installer-source-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(srcDir))
	}()

	binaryPath := filepath.Join(srcDir, "nmc-x86_64")
	require.NoError(t, os.WriteFile(binaryPath, binaryContents, fileio.NonExecutablePerms))

	destDir, err := os.MkdirTemp("", "eib-configurator-installer-dest-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(destDir))
	}()

	tests := []struct {
		name             string
		sourcePath       string
		installPath      string
		expectedContents []byte
		expectedError    string
	}{
		{
			name:          "Failure to copy non-existing binary",
			sourcePath:    "nmc-x86_64",
			expectedError: "copying file: opening source file: open nmc-x86_64: no such file or directory",
		},
		{
			name:             "Successfully installed binary",
			sourcePath:       binaryPath,
			installPath:      filepath.Join(destDir, "nmc"),
			expectedContents: binaryContents,
		},
	}

	var installer ConfiguratorInstaller

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = installer.InstallConfigurator(test.sourcePath, test.installPath)

			if test.expectedError != "" {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedError)
				return
			}

			require.NoError(t, err)

			contents, err := os.ReadFile(test.installPath)
			require.NoError(t, err)
			assert.Equal(t, test.expectedContents, contents)

			info, err := os.Stat(test.installPath)
			require.NoError(t, err)
			assert.Equal(t, fileio.ExecutablePerms, info.Mode())
		})
	}
}
