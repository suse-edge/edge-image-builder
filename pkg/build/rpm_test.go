package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRPMSourceDir(t *testing.T, addFiles bool) (tmpDir string, rpmSourceDir string, teardown func()) {
	tmpDir, err := os.MkdirTemp("", "eib-RPM-")
	require.NoError(t, err)

	rpmSourceDir = filepath.Join(tmpDir, "rpms")
	err = os.Mkdir(rpmSourceDir, 0o755)
	require.NoError(t, err)

	var file1 *os.File
	var file2 *os.File
	if addFiles {
		file1Path := filepath.Join(rpmSourceDir, "rpm1.rpm")
		file1, err = os.Create(file1Path)
		require.NoError(t, err)

		file2Path := filepath.Join(rpmSourceDir, "rpm2.rpm")
		file2, err = os.Create(file2Path)
		require.NoError(t, err)
	}

	return tmpDir, rpmSourceDir, func() {
		if addFiles {
			assert.NoError(t, file1.Close())
			assert.NoError(t, file2.Close())
		}
		assert.NoError(t, os.RemoveAll(tmpDir))
	}
}

func TestGetRPMFileNames(t *testing.T) {
	// Setup
	_, rpmSourceDir, teardown := setupRPMSourceDir(t, true)
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
	_, rpmSourceDir, teardown := setupRPMSourceDir(t, true)
	defer teardown()

	context, err := NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{context: context}

	// Test
	err = copyRPMs(rpmSourceDir, builder.context.CombustionDir, []string{"rpm1.rpm", "rpm2.rpm"})

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.context.CombustionDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.context.CombustionDir, "rpm2.rpm"))
	require.NoError(t, err)
}

func TestGetRPMFileNamesNoRPMs(t *testing.T) {
	// Setup
	_, rpmSourceDir, teardown := setupRPMSourceDir(t, false)
	defer teardown()

	// Test
	rpmFileNames, err := getRPMFileNames(rpmSourceDir)

	// Verify
	require.ErrorContains(t, err, "no RPMs found")

	assert.Empty(t, rpmFileNames)
}

func TestCopyRPMsNoRPMDestDir(t *testing.T) {
	// Setup
	_, rpmSourceDir, teardown := setupRPMSourceDir(t, true)
	defer teardown()

	// Test
	err := copyRPMs(rpmSourceDir, "", []string{"rpm1.rpm", "rpm2.rpm"})

	// Verify
	require.ErrorContains(t, err, "RPM destination directory cannot be empty")
}

func TestCopyRPMsNoRPMSrcDir(t *testing.T) {
	// Setup
	tmpDestDir, _, teardown := setupRPMSourceDir(t, true)
	defer teardown()

	// Test
	err := copyRPMs("", tmpDestDir, []string{"rpm1.rpm", "rpm2.rpm"})

	// Verify
	require.ErrorContains(t, err, "opening source file")
}

func TestWriteRPMScript(t *testing.T) {
	// Setup
	_, _, teardown := setupRPMSourceDir(t, true)
	defer teardown()

	context, err := NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{context: context}

	// Test
	err = builder.writeRPMScript([]string{"rpm1.rpm", "rpm2.rpm"})

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(builder.context.CombustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(modifyScriptMode), stats.Mode())

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "rpm1.rpm")
	assert.Contains(t, foundContents, "rpm2.rpm")
}

func TestProcessRPMs(t *testing.T) {
	// Setup
	tmpDir, _, teardown := setupRPMSourceDir(t, true)
	defer teardown()

	context, err := NewContext(tmpDir, "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{context: context}

	// Test
	err = builder.processRPMs()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.context.CombustionDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.context.CombustionDir, "rpm2.rpm"))
	require.NoError(t, err)

	expectedFilename := filepath.Join(builder.context.CombustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "rpm1.rpm")
	assert.Contains(t, foundContents, "rpm2.rpm")
}

func TestGenerateRPMPath(t *testing.T) {
	// Setup
	tmpDir, expectedPath, teardown := setupRPMSourceDir(t, false)
	defer teardown()

	context, err := NewContext(tmpDir, "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{context: context}

	// Test
	generatedPath, err := builder.generateRPMPath()

	// Verify
	require.NoError(t, err)

	assert.Equal(t, expectedPath, generatedPath)
}

func TestGenerateRPMPathNoRPMDir(t *testing.T) {
	// Setup
	tmpDir, _, teardown := setupRPMSourceDir(t, false)
	defer teardown()

	context, err := NewContext(tmpDir, "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{context: context}

	// Test
	_, err = builder.generateRPMPath()

	// Verify
	require.NoError(t, err)
}
