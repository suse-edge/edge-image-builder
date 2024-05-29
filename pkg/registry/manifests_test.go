package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestReadManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "sample-crd.yaml")

	// Test
	manifests, err := readManifest(manifestPath)

	// Verify
	require.NoError(t, err)

	// First manifest in sample-crd.yaml
	data := manifests[0]
	apiVersion, ok := data["apiVersion"].(string)
	require.True(t, ok)
	assert.Equal(t, "custom.example.com/v1", apiVersion)

	// Second manifest in sample-crd.yaml
	data = manifests[1]
	apiVersion, ok = data["apiVersion"].(string)
	require.True(t, ok)
	assert.Equal(t, "apps/v1", apiVersion)
}

func TestReadManifest_NoManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join()

	// Test
	_, err := readManifest(manifestPath)

	// Verify
	require.ErrorContains(t, err, "no such file or directory")
}

func TestReadManifest_InvalidManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "invalid-crd.yml")

	// Test
	_, err := readManifest(manifestPath)

	// Verify
	require.ErrorContains(t, err, "error unmarshalling manifest yaml")
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

	expectedImages := []string{"nginx:latest", "node:14", "custom-api:1.2.3", "mysql:5.7", "redis:6.0", "nginx:1.14.2"}

	// Test
	for _, manifest := range manifestData {
		storeManifestImages(manifest, extractedImagesSet)
	}
	allImages := make([]string, 0, len(extractedImagesSet))
	for uniqueImage := range extractedImagesSet {
		allImages = append(allImages, uniqueImage)
	}

	// Verify
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

func TestNew_InvalidManifestURL(t *testing.T) {
	// Setup
	buildDir := filepath.Join(os.TempDir(), "_manifests")
	require.NoError(t, os.MkdirAll(buildDir, os.ModePerm))
	defer func() {
		assert.NoError(t, os.RemoveAll(buildDir))
	}()

	ctx := &image.Context{
		BuildDir: buildDir,
		ImageDefinition: &image.Definition{
			Kubernetes: image.Kubernetes{
				Manifests: image.Manifests{
					URLs: []string{"k8s.io/examples/application/nginx-app.yaml"}},
			},
		},
	}

	// Test
	_, err := New(ctx, nil, "")

	// Verify
	require.Error(t, err)
	assert.ErrorContains(t, err, "downloading manifests: downloading manifest 'k8s.io/examples/application/nginx-app.yaml': executing request: Get \"k8s.io/examples/application/nginx-app.yaml\": unsupported protocol scheme \"\"")
}

func TestManifestImages_LocalManifestDirNotDefined(t *testing.T) {
	var registry Registry

	// Test
	containerImages, err := registry.ManifestImages()

	// Verify
	require.Error(t, err)
	assert.EqualError(t, err, "reading manifest dir: open : no such file or directory")
	assert.Empty(t, containerImages)
}

func TestManifestImages_InvalidLocalManifestsDir(t *testing.T) {
	// Setup
	registry := Registry{
		manifestsDir: "does-not-exist",
	}

	// Test
	_, err := registry.ManifestImages()

	// Verify
	require.ErrorContains(t, err, "reading manifest dir: open does-not-exist: no such file or directory")
}

func TestDownloadManifests_NoManifest(t *testing.T) {
	// Setup
	manifestDownloadDest := ""

	// Test
	manifestPaths, err := downloadManifests(nil, manifestDownloadDest)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 0, len(manifestPaths))
}

func TestDownloadManifests_InvalidURL(t *testing.T) {
	// Setup
	manifestURLs := []string{"k8s.io/examples/application/nginx-app.yaml"}
	manifestDownloadDest := ""

	// Test
	manifestPaths, err := downloadManifests(manifestURLs, manifestDownloadDest)

	// Verify
	require.ErrorContains(t, err, "downloading manifest 'k8s.io/examples/application/nginx-app.yaml': executing request: Get \"k8s.io/examples/application/nginx-app.yaml\": unsupported protocol scheme \"")
	assert.Equal(t, 0, len(manifestPaths))
}

func TestManifestImages_InvalidLocalManifest(t *testing.T) {
	// Setup
	const localManifestsSrcDir = "local-manifests"

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
	_, err := registry.ManifestImages()

	// Verify
	require.ErrorContains(t, err, "reading manifest: error unmarshalling manifest yaml")
}
