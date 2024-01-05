package network

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
		arch             image.Arch
		sourcePath       string
		installPath      string
		expectedContents []byte
		expectedError    string
	}{
		{
			name:          "Failure to copy non-existing binary",
			arch:          image.ArchTypeIntel,
			sourcePath:    "",
			expectedError: "copying file: opening source file: open nmc-x86_64: no such file or directory",
		},
		{
			name:             "Successfully installed x86_64 binary",
			arch:             image.ArchTypeIntel,
			sourcePath:       srcDir,
			installPath:      fmt.Sprintf("%s/nmc-amd", destDir),
			expectedContents: amdBinaryContents,
		},
		{
			name:             "Successfully installed aarch64 binary",
			arch:             image.ArchTypeARM,
			sourcePath:       srcDir,
			installPath:      fmt.Sprintf("%s/nmc-arm", destDir),
			expectedContents: armBinaryContents,
		},
	}

	var installer ConfiguratorInstaller

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = installer.InstallConfigurator(test.arch, test.sourcePath, test.installPath)

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
