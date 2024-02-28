package mount

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mountsConfName    = "mounts.conf"
	mountsConfContent = "/foo/bar:/foo/bar"
)

func TestDisableDefaultMountsRevertFunc(t *testing.T) {
	tests := []struct {
		name                     string
		dirName                  string
		overrideMountsConfExists bool
	}{
		{
			name:                     "Revert to the original mounts.conf override mount configuration",
			dirName:                  "disable-default-mounts-revert-to-existing-override-mounts-conf-",
			overrideMountsConfExists: true,
		},
		{
			name:    "Rever to the default mounts.conf configuration",
			dirName: "disable-default-mounts-revert-to-default-mounts-conf-",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", test.dirName)
			require.NoError(t, err)

			mountsConfPath := filepath.Join(dir, mountsConfName)
			if test.overrideMountsConfExists {
				mountsConfPath = createMountConf(t, dir)
			}

			revert, err := DisableDefaultMounts(mountsConfPath)
			require.NoError(t, err)
			err = revert()
			require.NoError(t, err)

			if test.overrideMountsConfExists {
				disabledMountsConfigPath := mountsConfPath + disableSuffix
				_, err = os.Stat(disabledMountsConfigPath)
				require.ErrorIs(t, err, fs.ErrNotExist)

				var content []byte
				content, err = os.ReadFile(mountsConfPath)
				require.NoError(t, err)
				assert.Equal(t, []byte(mountsConfContent), content)
			} else {
				_, err = os.Stat(mountsConfPath)
				require.ErrorIs(t, err, fs.ErrNotExist)
			}

			require.NoError(t, os.RemoveAll(dir))
		})
	}
}

func TestDisableDefaultMounts(t *testing.T) {
	tests := []struct {
		name                     string
		dirName                  string
		overrideMountsConfExists bool
	}{
		{
			name:                     "Replace existing mounts.conf file at mount override filepath",
			dirName:                  "disable-default-mounts-replace-existing-mounts-conf-",
			overrideMountsConfExists: true,
		},
		{
			name:    "Create new mounts.conf file at mount override filepath",
			dirName: "disable-default-mounts-create-new-mounts-conf-file-",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("", test.dirName)
			require.NoError(t, err)

			mountsConfPath := filepath.Join(dir, mountsConfName)
			if test.overrideMountsConfExists {
				mountsConfPath = createMountConf(t, dir)
			}

			_, err = DisableDefaultMounts(mountsConfPath)
			require.NoError(t, err)

			if test.overrideMountsConfExists {
				disabledMountsConfigPath := mountsConfPath + disableSuffix

				_, err = os.Stat(disabledMountsConfigPath)
				require.NoError(t, err)

				var content []byte
				content, err = os.ReadFile(disabledMountsConfigPath)
				require.NoError(t, err)
				assert.Equal(t, []byte(mountsConfContent), content)
			}

			mountsFile, err := os.Stat(mountsConfPath)
			require.NoError(t, err)
			assert.Equal(t, int64(0), mountsFile.Size())

			require.NoError(t, os.RemoveAll(dir))
		})
	}
}

func TestDisableDefaultMountsMissingMountPath(t *testing.T) {
	expectedErr := "creating empty /missing/file/path mount override file: open /missing/file/path: no such file or directory"

	_, err := DisableDefaultMounts("/missing/file/path")
	require.Error(t, err)
	assert.EqualError(t, err, expectedErr)
}

func createMountConf(t *testing.T, location string) string {
	mountsConf, err := os.Create(filepath.Join(location, mountsConfName))
	require.NoError(t, err)

	_, err = mountsConf.WriteString(mountsConfContent)
	require.NoError(t, err)

	err = mountsConf.Close()
	require.NoError(t, err)

	return mountsConf.Name()
}
