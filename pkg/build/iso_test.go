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

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			IsoConfiguration: image.IsoConfiguration{
				InstallDevice: "/dev/vda",
			},
		},
	}

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

	// Make sure that unattended mode is configured for GRUB
	assert.Contains(t, found, "set timeout=", "unattended mode is not configured properly in GRUB menu")

	// Make sure that target device is set as kernel cmdline argument
	assert.Contains(t, found, "rd.kiwi.oem.installdevice=/dev/vda", "install device target is not configured as kernel cmdline argument")

	// Make sure that the xorisso command also adds the grub.cfg mapping
	assert.Contains(t, found, "-map ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg /boot/grub2/grub.cfg", "xorisso doesn't have grub.cfg mapping")
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

func TestFindExtractedRawImage(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()
	builder := Builder{context: ctx}

	testExtractDir := filepath.Join(ctx.BuildDir, rawExtractDir)
	require.NoError(t, os.Mkdir(testExtractDir, os.FileMode(0o744)))

	testFiles := []string{
		"foo",
		"bar.raw",
		"baz",
	}
	for _, testFile := range testFiles {
		_, err := os.Create(filepath.Join(testExtractDir, testFile))
		require.NoError(t, err)
	}

	// Test
	foundFilename, err := builder.findExtractedRawImage()

	// Verify
	require.NoError(t, err)
	expectedFilename := filepath.Join(testExtractDir, "bar.raw")
	assert.Equal(t, expectedFilename, foundFilename)
}

func TestFindExtractedRawImage_NoRawImage(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()
	builder := Builder{context: ctx}

	testExtractDir := filepath.Join(ctx.BuildDir, rawExtractDir)
	require.NoError(t, os.Mkdir(testExtractDir, os.FileMode(0o744)))

	testFiles := []string{
		"foo",
		"baz",
	}
	for _, testFile := range testFiles {
		_, err := os.Create(filepath.Join(testExtractDir, testFile))
		require.NoError(t, err)
	}

	// Test
	foundFilename, err := builder.findExtractedRawImage()

	// Verify
	assert.Errorf(t, err, "unable to find a raw image")
	assert.Equal(t, "", foundFilename)
}
