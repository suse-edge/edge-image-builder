package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func setupRPMSourceDir(t *testing.T) (ctx *image.Context, rpmSourceDir string, teardown func()) {
	ctx, teardownCtx := setupContext(t)

	rpmSourceDir = filepath.Join(ctx.ImageConfigDir, "rpms")
	require.NoError(t, os.Mkdir(rpmSourceDir, 0o755))

	file1, err := os.Create(filepath.Join(rpmSourceDir, "rpm1.rpm"))
	require.NoError(t, err)

	file2, err := os.Create(filepath.Join(rpmSourceDir, "rpm2.rpm"))
	require.NoError(t, err)

	return ctx, rpmSourceDir, func() {
		assert.NoError(t, file1.Close())
		assert.NoError(t, file2.Close())

		teardownCtx()
	}
}

func TestGetRPMFileNames(t *testing.T) {
	// Setup
	_, rpmSourceDir, teardown := setupRPMSourceDir(t)
	defer teardown()

	// Test
	rpmFileNames, err := getRPMFileNames(rpmSourceDir)

	// Verify
	require.NoError(t, err)

	assert.Contains(t, rpmFileNames, "rpm1.rpm")
	assert.Contains(t, rpmFileNames, "rpm2.rpm")
}

func TestCopyRPMs(t *testing.T) {
	// Setup
	ctx, rpmSourceDir, teardown := setupRPMSourceDir(t)
	defer teardown()

	tmpDestDir := filepath.Join(ctx.BuildDir, "temp-dest-dir")
	require.NoError(t, os.Mkdir(tmpDestDir, 0o755))

	// Test
	require.NoError(t, copyRPMs(rpmSourceDir, tmpDestDir, []string{"rpm1.rpm", "rpm2.rpm"}))

	// Verify
	_, err := os.Stat(filepath.Join(tmpDestDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(tmpDestDir, "rpm2.rpm"))
	require.NoError(t, err)
}

func TestGetRPMFileNamesNoRPMs(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-rpm-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	// Test
	rpmFileNames, err := getRPMFileNames(tmpDir)

	// Verify
	require.ErrorContains(t, err, "no RPMs found")

	assert.Empty(t, rpmFileNames)
}

func TestCopyRPMsNoRPMDestDir(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-rpm-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	// Test
	err = copyRPMs(tmpDir, "", []string{"rpm1.rpm", "rpm2.rpm"})

	// Verify
	require.ErrorContains(t, err, "RPM destination directory cannot be empty")
}

func TestCopyRPMsNoRPMSrcDir(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-rpm-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmpDir))
	}()

	// Test
	err = copyRPMs("", tmpDir, []string{"rpm1.rpm", "rpm2.rpm"})

	// Verify
	require.ErrorContains(t, err, "opening source file")
}

func TestWriteRPMScript(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	script, err := writeRPMScript(ctx, []string{"rpm1.rpm", "rpm2.rpm"})

	// Verify
	require.NoError(t, err)

	assert.Equal(t, modifyRPMScriptName, script)

	expectedFilename := filepath.Join(ctx.CombustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "rpm1.rpm")
	assert.Contains(t, foundContents, "rpm2.rpm")
}

func TestConfigureRPMs(t *testing.T) {
	// Setup
	ctx, _, teardown := setupRPMSourceDir(t)
	defer teardown()

	// Test
	scripts, err := configureRPMs(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, modifyRPMScriptName, scripts[0])

	_, err = os.Stat(filepath.Join(ctx.CombustionDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(ctx.CombustionDir, "rpm2.rpm"))
	require.NoError(t, err)

	expectedFilename := filepath.Join(ctx.CombustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "rpm1.rpm")
	assert.Contains(t, foundContents, "rpm2.rpm")
}
