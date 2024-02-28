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

func TestDisableDefaultMountsRevertToOverridelMountsFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "disable-default-mounts-revert-to-override-mounts-file-")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	mountsConfPath := createMountConf(t, dir)

	revert, err := DisableDefaultMounts(mountsConfPath)
	require.NoError(t, err)
	err = revert()
	require.NoError(t, err)

	disabledMountsConfigPath := mountsConfPath + disableSuffix
	_, err = os.Stat(disabledMountsConfigPath)
	require.ErrorIs(t, err, fs.ErrNotExist)

	var content []byte
	content, err = os.ReadFile(mountsConfPath)
	require.NoError(t, err)
	assert.Equal(t, []byte(mountsConfContent), content)
}

func TestDisableDefaultMountsRevertToDefaultMountsFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "disable-default-mounts-revert-to-default-mounts-file-")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	mountsConfPath := filepath.Join(dir, mountsConfName)

	revert, err := DisableDefaultMounts(mountsConfPath)
	require.NoError(t, err)
	err = revert()
	require.NoError(t, err)

	_, err = os.Stat(mountsConfPath)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestDisableDefaultMountsExistingOverrideMountsFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "disable-default-mounts-existing-override-mounts-file-")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	mountsConfPath := createMountConf(t, dir)

	_, err = DisableDefaultMounts(mountsConfPath)
	require.NoError(t, err)

	disabledMountsConfigPath := mountsConfPath + disableSuffix

	_, err = os.Stat(disabledMountsConfigPath)
	require.NoError(t, err)

	var content []byte
	content, err = os.ReadFile(disabledMountsConfigPath)
	require.NoError(t, err)
	assert.Equal(t, []byte(mountsConfContent), content)

	mountsFile, err := os.Stat(mountsConfPath)
	require.NoError(t, err)
	assert.Equal(t, int64(0), mountsFile.Size())
}

func TestDisableDefaultMountsNoExistingOverrideMountsFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "disable-default-mounts-no-existing-override-mounts-file-")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	mountsConfPath := filepath.Join(dir, mountsConfName)

	_, err = DisableDefaultMounts(mountsConfPath)
	require.NoError(t, err)

	mountsFile, err := os.Stat(mountsConfPath)
	require.NoError(t, err)
	assert.Equal(t, int64(0), mountsFile.Size())
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
