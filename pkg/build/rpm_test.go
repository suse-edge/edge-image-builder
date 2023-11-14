package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRPMFileNames(t *testing.T) {
	// Setup
	context, err := NewContext("../config/testdata", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := &Builder{
		context: context,
	}

	RPMSourceDir, err := builder.generateRPMPath()
	require.NoError(t, err)

	file1Path := filepath.Join(RPMSourceDir, "rpm1.rpm")
	file1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer os.Remove(file1Path)

	file2Path := filepath.Join(RPMSourceDir, "rpm2.rpm")
	file2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer os.Remove(file2Path)

	// Test
	RPMFileNames, err := getRPMFileNames(RPMSourceDir)

	// Verify
	require.NoError(t, err)

	assert.Contains(t, RPMFileNames, "rpm1.rpm")
	assert.Contains(t, RPMFileNames, "rpm2.rpm")

	// Cleanup
	assert.NoError(t, file1.Close())
	assert.NoError(t, file2.Close())
}

func TestCopyRPMs(t *testing.T) {
	// Setup
	context, err := NewContext("../config/testdata", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := &Builder{
		context: context,
	}

	RPMDestDir := builder.combustionDir
	RPMSourceDir, err := builder.generateRPMPath()
	require.NoError(t, err)

	file1Path := filepath.Join(RPMSourceDir, "rpm1.rpm")
	file1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer os.Remove(file1Path)

	file2Path := filepath.Join(RPMSourceDir, "rpm2.rpm")
	file2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer os.Remove(file2Path)

	RPMFileNames, err := getRPMFileNames(RPMSourceDir)
	require.NoError(t, err)

	// Test
	err = copyRPMs(RPMSourceDir, RPMDestDir, RPMFileNames)

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(RPMDestDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(RPMDestDir, "rpm2.rpm"))
	require.NoError(t, err)

	// Cleanup
	assert.NoError(t, file1.Close())
	assert.NoError(t, file2.Close())
}

func TestGetRPMFileNamesNoRPMs(t *testing.T) {
	// Setup
	context, err := NewContext("../config/testdata", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := &Builder{
		context: context,
	}

	RPMSourceDir, err := builder.generateRPMPath()
	require.NoError(t, err)

	// Test
	RPMFileNames, err := getRPMFileNames(RPMSourceDir)

	// Verify
	require.ErrorContains(t, err, "no rpms found")

	assert.Empty(t, RPMFileNames)
}

func TestCopyRPMsNoRPMDir(t *testing.T) {
	// Setup
	context, err := NewContext("../config/ThisDirDoesNotExist", "", true)
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	RPMDestDir := builder.combustionDir
	RPMSourceDir, err := builder.generateRPMPath()
	require.NoError(t, err)

	// Test
	err = copyRPMs(RPMSourceDir, RPMDestDir, nil)

	// Verify
	require.NoError(t, err)
}

func TestWriteRPMScript(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	buildConfig := config.BuildConfig{
		ImageConfigDir: "../config/testdata",
	}
	builder := New(nil, &buildConfig)
	require.NoError(t, builder.prepareBuildDir())

	RPMSourceDir, err := builder.generateRPMPath()
	require.NoError(t, err)

	file1Path := filepath.Join(RPMSourceDir, "rpm1.rpm")
	file1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer os.Remove(file1Path)

	file2Path := filepath.Join(RPMSourceDir, "rpm2.rpm")
	file2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer os.Remove(file2Path)

	RPMFileNames, err := getRPMFileNames(RPMSourceDir)
	require.NoError(t, err)

	// Test
	err = builder.writeRPMScript(RPMFileNames)

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(builder.combustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(modifyScriptMode), stats.Mode())

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "rpm1.rpm")
	assert.Contains(t, foundContents, "rpm2.rpm")

	// Cleanup
	assert.NoError(t, file1.Close())
	assert.NoError(t, file2.Close())
}

func TestProcessRPMs(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		ImageConfigDir: "../config/testdata",
	}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	RPMDestDir := builder.combustionDir
	RPMSourceDir, err := builder.generateRPMPath()
	require.NoError(t, err)

	file1Path := filepath.Join(RPMSourceDir, "rpm1.rpm")
	file1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer os.Remove(file1Path)

	file2Path := filepath.Join(RPMSourceDir, "rpm2.rpm")
	file2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer os.Remove(file2Path)

	// Test
	err = builder.processRPMs()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(RPMDestDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(RPMDestDir, "rpm2.rpm"))
	require.NoError(t, err)

	expectedFilename := filepath.Join(RPMDestDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "rpm1.rpm")
	assert.Contains(t, foundContents, "rpm2.rpm")

	// Cleanup
	assert.NoError(t, file1.Close())
	assert.NoError(t, file2.Close())
}

func TestGenerateRPMPath(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		ImageConfigDir: "../config/testdata",
	}
	builder := New(nil, &bc)

	expectedPath := filepath.Join(builder.buildConfig.ImageConfigDir, "rpms")

	// Test
	generatedPath, err := builder.generateRPMPath()

	// Verify
	require.NoError(t, err)

	assert.Equal(t, expectedPath, generatedPath)
}

func TestGenerateRPMPathNoRPMDir(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	builder := New(nil, &bc)

	// Test
	_, err := builder.generateRPMPath()

	// Verify
	require.NoError(t, err)
}
