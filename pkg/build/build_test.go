package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestPrepareBuildDir(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	builder := New(nil, &bc)

	// Test
	err := builder.prepareBuildDir()
	defer os.RemoveAll(builder.eibBuildDir)

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(builder.eibBuildDir)
	require.NoError(t, err)
}

func TestPrepareBuildDirExistingDir(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	bc := config.BuildConfig{BuildDir: tmpDir}
	builder := New(nil, &bc)

	// Test
	err = builder.prepareBuildDir()

	// Verify
	require.NoError(t, err)
	require.Equal(t, tmpDir, builder.eibBuildDir)
}

func TestCleanUpBuildDirTrue(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	bc := config.BuildConfig{
		BuildDir: tmpDir,
		DeleteBuildDir: true,
	}
	builder := New(nil, &bc)
	builder.prepareBuildDir()

	// Test
	err = builder.cleanUpBuildDir()

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(tmpDir)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestCleanUpBuildDirFalse(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	bc := config.BuildConfig{
		BuildDir: tmpDir,
		DeleteBuildDir: false,
	}
	builder := New(nil, &bc)
	builder.prepareBuildDir()

	// Test
	err = builder.cleanUpBuildDir()

	// Verify
	require.NoError(t, err)
	_, err = os.Stat(tmpDir)
	require.NoError(t, err)
}

func TestGenerateCombustionScript(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.RemoveAll(builder.eibBuildDir)

	builder.combustionScripts = append(builder.combustionScripts, "foo.sh", "bar.sh")

	// Test
	err = builder.generateCombustionScript()

	// Verify
	require.NoError(t, err)

	scriptBytes, err := os.ReadFile(filepath.Join(builder.combustionDir, "script"))
	require.NoError(t, err)
	scriptData := string(scriptBytes)
	assert.Contains(t, scriptData, "#!/bin/bash")
	assert.Contains(t, scriptData, "foo.sh")
	assert.Contains(t, scriptData, "bar.sh")
}

func TestWriteCombustionFile(t *testing.T) {
	// Setup
	builder := New(nil, &config.BuildConfig{})
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.RemoveAll(builder.eibBuildDir)

	testData := "Edge Image Builder"
	testFilename := "combustion-file.sh"

	// Test
	err = builder.writeCombustionFile(testFilename, testData, nil)

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(builder.combustionDir, testFilename)
	foundData, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, testData, string(foundData))

	// Make sure the file isn't automatically added to the combustion scripts list
	require.Equal(t, 0, len(builder.combustionScripts))
}

func TestWriteBuildDirFile(t *testing.T) {
	// Setup
	builder := New(nil, &config.BuildConfig{})
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.RemoveAll(builder.eibBuildDir)

	testData := "Edge Image Builder"
	testFilename := "build-dir-file.sh"

	// Test
	err = builder.writeBuildDirFile(testFilename, testData, nil)

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(builder.eibBuildDir, testFilename)
	foundData, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, testData, string(foundData))
}

func TestWriteFileWithTemplate(t *testing.T) {
	// Setup
	builder := New(nil, &config.BuildConfig{})

	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testData := "{{.Foo}} and {{.Bar}}"
	values := struct {
		Foo string
		Bar string
	}{
		Foo: "ooF",
		Bar: "raB",
	}
	testFilename := filepath.Join(tmpDir, "write-file-with-template.sh")

	// Test
	err = builder.writeFile(testFilename, testData, &values)

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(builder.eibBuildDir, testFilename)
	foundData, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, "ooF and raB", string(foundData))
}