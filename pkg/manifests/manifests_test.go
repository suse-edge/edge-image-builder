package manifests

import (
	"os"
	"path/filepath"
	"sort"
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

func TestReadManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "sample-crd.yaml")

	// Test
	manifestData, err := readManifest(manifestPath)

	// Verify
	require.NoError(t, err)

	data, ok := manifestData.(map[string]interface{})
	require.True(t, ok)

	apiVersion, ok := data["apiVersion"].(string)
	require.True(t, ok)

	assert.Equal(t, "custom.example.com/v1", apiVersion)
}

func TestReadManifestNoManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join()

	// Test
	_, err := readManifest(manifestPath)

	// Verify
	require.ErrorContains(t, err, "no such file or directory")
}

func TestReadManifestInvalidManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "invalid-crd.yaml")

	// Test
	_, err := readManifest(manifestPath)

	// Verify
	require.NoError(t, err)
}

func TestReadManifestEmptyManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "empty-crd.yaml")

	// Test
	_, err := readManifest(manifestPath)

	// Verify
	assert.Error(t, err, "invalid manifest")
}

func TestFindImagesInManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "sample-crd.yaml")
	manifestData, err := readManifest(manifestPath)
	require.NoError(t, err)

	expectedImages := []string{"nginx:latest", "node:14", "custom-api:1.2.3", "mysql:5.7", "redis:6.0"}
	sort.Strings(expectedImages)

	// Test
	foundImages, err := findImagesInManifest(manifestData)
	sort.Strings(foundImages)

	// Verify
	assert.Equal(t, expectedImages, foundImages)
}

func TestFindImagesInManifestEmptyManifest(t *testing.T) {
	// Setup
	var manifestData interface{}

	// Test
	foundImages, err := findImagesInManifest(manifestData)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, []string{}, foundImages)
}

func TestFindImagesInManifestInvalidManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "invalid-crd.yaml")
	manifestData, err := readManifest(manifestPath)
	require.NoError(t, err)

	// Test
	foundImages, err := findImagesInManifest(manifestData)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, []string{}, foundImages)
}

func TestDownloadManifests(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	foundImages, err := findImagesInManifest(manifestData)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, []string{}, foundImages)
}
