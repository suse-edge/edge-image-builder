package build

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func setupContext(t *testing.T) (ctx *image.Context, teardown func()) {
	// Copied from combustion_test due to time. This should eventually be refactored
	// to something cleaner.

	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	buildDir, err := os.MkdirTemp("", "eib-build-")
	require.NoError(t, err)

	combustionDir, err := os.MkdirTemp("", "eib-combustion-")
	require.NoError(t, err)

	ctx = &image.Context{
		ImageConfigDir:  configDir,
		BuildDir:        buildDir,
		CombustionDir:   combustionDir,
		ImageDefinition: &image.Definition{},
	}

	return ctx, func() {
		assert.NoError(t, os.RemoveAll(combustionDir))
		assert.NoError(t, os.RemoveAll(buildDir))
		assert.NoError(t, os.RemoveAll(configDir))
	}
}

func TestDeleteNoExistingImage(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	builder := Builder{
		context: &image.Context{
			ImageConfigDir: tmpDir,
			ImageDefinition: &image.Definition{
				Image: image.Image{
					OutputImageName: "not-there",
				},
			},
		},
	}

	// Test
	err = builder.deleteExistingOutputIso()

	// Verify
	require.NoError(t, err)
}

func TestDeleteExistingImage(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	builder := Builder{
		context: &image.Context{
			ImageConfigDir: tmpDir,
			ImageDefinition: &image.Definition{
				Image: image.Image{
					OutputImageName: "not-there",
				},
			},
		},
	}

	_, err = os.Create(builder.generateOutputImageFilename())
	require.NoError(t, err)

	// Test
	err = builder.deleteExistingOutputIso()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(builder.generateOutputImageFilename())
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestWriteIsoScript_Extract(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()
	builder := Builder{context: ctx}

	// Test
	err := builder.writeIsoScript(extractIsoTemplate, extractIsoScriptName)

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(ctx.BuildDir, extractIsoScriptName)
	_, err = os.Stat(expectedFilename)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)
	found := string(foundBytes)

	expectedIsoExtractDir := filepath.Join(ctx.BuildDir, isoExtractDir)
	assert.Contains(t, found, fmt.Sprintf("ISO_EXTRACT_DIR=%s", expectedIsoExtractDir))

	expectedRawExtractDir := filepath.Join(ctx.BuildDir, rawExtractDir)
	assert.Contains(t, found, fmt.Sprintf("RAW_EXTRACT_DIR=%s", expectedRawExtractDir))

	expectedIsoPath := builder.generateBaseImageFilename()
	assert.Contains(t, found, fmt.Sprintf("ISO_SOURCE=%s", expectedIsoPath))
}

func TestWriteIsoScript_Rebuild(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()
	builder := Builder{context: ctx}

	// Test
	err := builder.writeIsoScript(rebuildIsoTemplate, rebuildIsoScriptName)

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(ctx.BuildDir, rebuildIsoScriptName)
	_, err = os.Stat(expectedFilename)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)
	found := string(foundBytes)

	expectedIsoExtractDir := filepath.Join(ctx.BuildDir, isoExtractDir)
	assert.Contains(t, found, fmt.Sprintf("ISO_EXTRACT_DIR=%s", expectedIsoExtractDir))

	expectedRawExtractDir := filepath.Join(ctx.BuildDir, rawExtractDir)
	assert.Contains(t, found, fmt.Sprintf("RAW_EXTRACT_DIR=%s", expectedRawExtractDir))

	expectedIsoPath := builder.generateBaseImageFilename()
	assert.Contains(t, found, fmt.Sprintf("ISO_SOURCE=%s", expectedIsoPath))

	expectedOutputImage := builder.generateOutputImageFilename()
	assert.Contains(t, found, fmt.Sprintf("OUTPUT_IMAGE=%s", expectedOutputImage))

	expectedCombustionDir := ctx.CombustionDir
	assert.Contains(t, found, fmt.Sprintf("COMBUSTION_DIR=%s", expectedCombustionDir))
}

func TestCreateIsoCommand(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()
	builder := Builder{context: ctx}

	// Test
	cmd, logFile, err := builder.createIsoCommand("test-log", "test-script")

	// Verify
	require.NoError(t, err)
	require.NotNil(t, cmd)

	expectedCommandPath := filepath.Join(ctx.BuildDir, "test-script")
	assert.Equal(t, expectedCommandPath, cmd.Path)
	assert.Equal(t, logFile, cmd.Stdout)
	assert.Equal(t, logFile, cmd.Stderr)
}
