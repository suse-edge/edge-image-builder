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
	tmpDir, err := os.MkdirTemp("", "eib-get-RPM-file-names-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	RPMSourceDir := filepath.Join(tmpDir, "rpms")
	err = os.Mkdir(RPMSourceDir, 0755)
	require.NoError(t, err)

	bc := config.BuildConfig{}
	builder := New(nil, &bc)
	err = builder.prepareBuildDir()
	require.NoError(t, err)

	file1Path := filepath.Join(RPMSourceDir, "rpm1.rpm")
	file1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer file1.Close()

	file2Path := filepath.Join(RPMSourceDir, "rpm2.rpm")
	file2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer file2.Close()

	// Test
	RPMFileNames, err := getRPMFileNames(RPMSourceDir)

	// Verify
	require.NoError(t, err)

	assert.Contains(t, RPMFileNames, "rpm1.rpm")
	assert.Contains(t, RPMFileNames, "rpm2.rpm")
}

func TestCopyRPMs(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-copy-RPMs-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	RPMSourceDir := filepath.Join(tmpDir, "rpms")
	err = os.Mkdir(RPMSourceDir, 0755)
	require.NoError(t, err)

	bc := config.BuildConfig{}
	builder := New(nil, &bc)
	err = builder.prepareBuildDir()
	require.NoError(t, err)

	file1Path := filepath.Join(RPMSourceDir, "rpm1.rpm")
	file1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer file1.Close()

	file2Path := filepath.Join(RPMSourceDir, "rpm2.rpm")
	file2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer file2.Close()

	RPMFileNames, err := getRPMFileNames(RPMSourceDir)
	require.NoError(t, err)

	// Test
	err = copyRPMs(RPMSourceDir, builder.combustionDir, RPMFileNames)

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, "rpm2.rpm"))
	require.NoError(t, err)
}

func TestGetRPMFileNamesNoRPMs(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-copy-RPMs-test-no-RPMs")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	RPMSourceDir := filepath.Join(tmpDir, "rpms")
	err = os.Mkdir(RPMSourceDir, 0755)
	require.NoError(t, err)

	bc := config.BuildConfig{}
	builder := New(nil, &bc)
	err = builder.prepareBuildDir()
	require.NoError(t, err)

	// Test
	RPMFileNames, err := getRPMFileNames(RPMSourceDir)

	// Verify
	require.ErrorContains(t, err, "no RPMs found")

	assert.Empty(t, RPMFileNames)
}

func TestCopyRPMsNoRPMDir(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-copy-RPMs-test-no-dir-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	bc := config.BuildConfig{}
	builder := New(nil, &bc)
	err = builder.prepareBuildDir()
	require.NoError(t, err)

	RPMSourceDir, err := builder.generateRPMPath()
	require.NoError(t, err)

	// Test
	err = copyRPMs(RPMSourceDir, builder.combustionDir, nil)

	// Verify
	require.NoError(t, err)
}

func TestWriteRPMScript(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-write-RPM-script-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	RPMSourceDir := filepath.Join(tmpDir, "rpms")
	err = os.Mkdir(RPMSourceDir, 0755)
	require.NoError(t, err)

	buildConfig := config.BuildConfig{}
	builder := New(nil, &buildConfig)
	require.NoError(t, builder.prepareBuildDir())

	file1Path := filepath.Join(RPMSourceDir, "rpm1.rpm")
	file1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer file1.Close()

	file2Path := filepath.Join(RPMSourceDir, "rpm2.rpm")
	file2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer file2.Close()

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
}

func TestProcessRPMs(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-process-RPMs-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	RPMSourceDir := filepath.Join(tmpDir, "rpms")
	err = os.Mkdir(RPMSourceDir, 0755)
	require.NoError(t, err)

	bc := config.BuildConfig{
		ImageConfigDir: tmpDir,
	}
	builder := New(nil, &bc)
	err = builder.prepareBuildDir()
	require.NoError(t, err)

	file1Path := filepath.Join(RPMSourceDir, "rpm1.rpm")
	file1, err := os.Create(file1Path)
	require.NoError(t, err)
	defer file1.Close()

	file2Path := filepath.Join(RPMSourceDir, "rpm2.rpm")
	file2, err := os.Create(file2Path)
	require.NoError(t, err)
	defer file2.Close()

	// Test
	err = builder.processRPMs()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, "rpm2.rpm"))
	require.NoError(t, err)

	expectedFilename := filepath.Join(builder.combustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	foundContents := string(foundBytes)
	assert.Contains(t, foundContents, "rpm1.rpm")
	assert.Contains(t, foundContents, "rpm2.rpm")
}

func TestGenerateRPMPath(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-generate-RPM-path-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	RPMSourceDir := filepath.Join(tmpDir, "rpms")
	err = os.Mkdir(RPMSourceDir, 0755)
	require.NoError(t, err)

	bc := config.BuildConfig{
		ImageConfigDir: tmpDir,
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
