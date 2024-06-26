package combustion

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func setupOsFilesConfigDir(t *testing.T, empty bool) (ctx *image.Context, teardown func()) {
	ctx, teardown = setupContext(t)

	testOsFilesDir := filepath.Join(ctx.ImageConfigDir, osFilesConfigDir)
	err := os.Mkdir(testOsFilesDir, 0o755)
	require.NoError(t, err)

	if !empty {
		nestedOsFilesDir := filepath.Join(testOsFilesDir, "etc", "ssh")
		err = os.MkdirAll(nestedOsFilesDir, 0o755)
		require.NoError(t, err)

		testFile := filepath.Join(nestedOsFilesDir, "test-config-file")
		_, err = os.Create(testFile)
		require.NoError(t, err)
	}

	return
}

func TestConfigureOSFiles(t *testing.T) {
	// Setup
	ctx, teardown := setupOsFilesConfigDir(t, false)
	defer teardown()

	// Test
	scriptNames, err := configureOSFiles(ctx)

	// Verify
	require.NoError(t, err)

	assert.Equal(t, []string{osFilesScriptName}, scriptNames)

	// -- Combustion Script
	expectedCombustionScript := filepath.Join(ctx.CombustionDir, osFilesScriptName)
	contents, err := os.ReadFile(expectedCombustionScript)
	require.NoError(t, err)
	assert.Contains(t, string(contents), "cp -R")

	// -- Files
	expectedFile := filepath.Join(ctx.CombustionDir, osFilesConfigDir, "etc", "ssh", "test-config-file")
	assert.FileExists(t, expectedFile)
}

func TestConfigureOSFiles_EmptyDirectory(t *testing.T) {
	// Setup
	ctx, teardown := setupOsFilesConfigDir(t, true)
	defer teardown()

	// Test
	scriptName, err := configureOSFiles(ctx)

	// Verify
	assert.Nil(t, scriptName)

	srcDirectory := filepath.Join(ctx.ImageConfigDir, osFilesConfigDir)
	assert.EqualError(t, err, fmt.Sprintf("no files found in directory %s", srcDirectory))
}
