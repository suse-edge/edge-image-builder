package network

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

func TestConfiguratorInstaller_InstallConfigurator(t *testing.T) {
	amdBinaryContents := []byte("amd")
	armBinaryContents := []byte("arm")

	srcDir, err := os.MkdirTemp("", "eib-configurator-installer-source-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(srcDir))
	}()

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "nmc-x86_64"), amdBinaryContents, fileio.NonExecutablePerms))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "nmc-aarch64"), armBinaryContents, fileio.NonExecutablePerms))

	destDir, err := os.MkdirTemp("", "eib-configurator-installer-dest-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(destDir))
	}()

	tests := []struct {
		name             string
		imageName        string
		sourcePath       string
		installPath      string
		expectedContents []byte
		expectedError    string
	}{
		{
			name:          "Failure to detect architecture from image name",
			imageName:     "abc",
			expectedError: "failed to determine arch of image abc",
		},
		{
			name:          "Failure to copy non-existing binary",
			imageName:     "abc-x86_64",
			sourcePath:    "",
			expectedError: "copying file: opening source file: open nmc-x86_64: no such file or directory",
		},
		{
			name:             "Successfully installed x86_64 binary",
			imageName:        "abc-x86_64",
			sourcePath:       srcDir,
			installPath:      fmt.Sprintf("%s/nmc-amd", destDir),
			expectedContents: amdBinaryContents,
		},
		{
			name:             "Successfully installed aarch64 binary",
			imageName:        "abc-aarch64",
			sourcePath:       srcDir,
			installPath:      fmt.Sprintf("%s/nmc-arm", destDir),
			expectedContents: armBinaryContents,
		},
	}

	var installer ConfiguratorInstaller

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = installer.InstallConfigurator(test.imageName, test.sourcePath, test.installPath)

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
