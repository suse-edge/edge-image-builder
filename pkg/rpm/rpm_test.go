package rpm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRPMSourceDir(t *testing.T) (dirPath string, teardown func()) {
	dirPath = filepath.Join(os.TempDir(), "rpms")
	require.NoError(t, os.Mkdir(dirPath, 0o755))

	file1, err := os.Create(filepath.Join(dirPath, "rpm1.rpm"))
	require.NoError(t, err)

	file2, err := os.Create(filepath.Join(dirPath, "rpm2.rpm"))
	require.NoError(t, err)

	return dirPath, func() {
		assert.NoError(t, file1.Close())
		assert.NoError(t, file2.Close())
		assert.NoError(t, os.RemoveAll(dirPath))
	}
}

func TestGetRPMFileNames(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copy-location")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	rpmSourceDir, teardown := setupRPMSourceDir(t)
	defer teardown()

	rpmFileNames, err := CopyRPMs(rpmSourceDir, tmpDir)

	require.NoError(t, err)

	require.Len(t, rpmFileNames, 2)
	assert.Contains(t, rpmFileNames, "rpm1.rpm")
	assert.Contains(t, rpmFileNames, "rpm2.rpm")
}

func TestCopyRPMs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copy-location")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	rpmSourceDir, teardown := setupRPMSourceDir(t)
	defer teardown()

	_, err = CopyRPMs(rpmSourceDir, tmpDir)

	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDir, "rpm2.rpm"))
	require.NoError(t, err)
}

func TestGetRPMFileNamesNoRPMs(t *testing.T) {
	fromTempDir, err := os.MkdirTemp("", "from-location")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(fromTempDir))
	}()

	toTempDir, err := os.MkdirTemp("", "to-location")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(toTempDir))
	}()

	// pass the same dir, to simulate a missing
	rpmFileNames, err := CopyRPMs(fromTempDir, toTempDir)

	require.NoError(t, err)
	assert.Empty(t, rpmFileNames)
}

func TestCopyRPMsNoRPMDestDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copy-location")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	_, err = CopyRPMs(tmpDir, "")

	require.Error(t, err)
	require.ErrorContains(t, err, "RPM destination directory cannot be empty")
}

func TestCopyRPMsNoRPMSrcDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copy-location")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	_, err = CopyRPMs("", tmpDir)

	require.Error(t, err)
	require.ErrorContains(t, err, "reading RPM source dir")
}
