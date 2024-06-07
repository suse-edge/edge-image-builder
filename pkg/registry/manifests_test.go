package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

func TestReadManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "sample-crd.yaml")

	// Test
	resources, err := readManifest(manifestPath)

	// Verify
	require.NoError(t, err)
	require.Len(t, resources, 2)

	// First resource in sample-crd.yaml
	r := resources[0]
	apiVersion, ok := r["apiVersion"].(string)
	require.True(t, ok)
	assert.Equal(t, "custom.example.com/v1", apiVersion)

	// Second resource in sample-crd.yaml
	r = resources[1]
	apiVersion, ok = r["apiVersion"].(string)
	require.True(t, ok)
	assert.Equal(t, "apps/v1", apiVersion)
}

func TestReadManifest_NoManifest(t *testing.T) {
	_, err := readManifest("")
	require.ErrorContains(t, err, "no such file or directory")
}

func TestReadManifest_InvalidManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "invalid-crd.yml")

	// Test
	_, err := readManifest(manifestPath)

	// Verify
	require.ErrorContains(t, err, "unmarshalling manifest")
}

func TestReadManifest_EmptyManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "empty-crd.yaml")

	// Test
	_, err := readManifest(manifestPath)

	// Verify
	assert.Error(t, err, "invalid manifest")
}

func TestStoreManifestImages(t *testing.T) {
	// Setup
	var extractedImagesSet = make(map[string]bool)
	manifestPath := filepath.Join("testdata", "sample-crd.yaml")
	manifestData, err := readManifest(manifestPath)
	require.NoError(t, err)

	// Test
	for _, manifest := range manifestData {
		storeManifestImages(manifest, extractedImagesSet)
	}
	allImages := make([]string, 0, len(extractedImagesSet))
	for uniqueImage := range extractedImagesSet {
		allImages = append(allImages, uniqueImage)
	}

	// Verify
	expectedImages := []string{"nginx:latest", "node:14", "custom-api:1.2.3", "mysql:5.7", "redis:6.0", "nginx:1.14.2"}
	assert.ElementsMatch(t, expectedImages, allImages)
}

func TestStoreManifestImages_InvalidKinds(t *testing.T) {
	// Setup
	var extractedImagesSet = make(map[string]bool)
	manifestData := map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "InvalidKind",
		"spec": map[string]any{
			"containers": []any{
				map[string]any{
					"name":  "nginx",
					"image": "nginx:1.14.2",
				},
			},
		},
	}

	// Test
	storeManifestImages(manifestData, extractedImagesSet)

	// Verify
	assert.Equal(t, map[string]bool{}, extractedImagesSet)
}

func TestStoreManifestImages_EmptyManifest(t *testing.T) {
	// Setup
	var extractedImagesSet = make(map[string]bool)
	var manifestData map[string]any

	// Test
	storeManifestImages(manifestData, extractedImagesSet)

	// Verify
	assert.Equal(t, map[string]bool{}, extractedImagesSet)
}

func TestManifestImages_InvalidLocalManifest(t *testing.T) {
	// Setup
	localManifestsSrcDir := "local-manifests"

	require.NoError(t, os.Mkdir(localManifestsSrcDir, 0o755))
	defer func() {
		assert.NoError(t, os.RemoveAll(localManifestsSrcDir))
	}()

	sourceManifest := filepath.Join("testdata", "invalid-crd.yml")
	destinationManifest := filepath.Join(localManifestsSrcDir, "invalid-crd.yml")
	require.NoError(t, fileio.CopyFile(sourceManifest, destinationManifest, fileio.NonExecutablePerms))

	registry := Registry{
		manifestsDir: localManifestsSrcDir,
	}

	// Test
	_, err := registry.manifestImages()

	// Verify
	require.ErrorContains(t, err, "reading manifest 'local-manifests/invalid-crd.yml': unmarshalling manifest")
}
