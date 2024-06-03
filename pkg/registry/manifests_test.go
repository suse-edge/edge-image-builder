package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

const (
	localManifestsSrcDir = "local-manifests"
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

func TestGetManifestPaths(t *testing.T) {
	// Setup
	manifestSrcDir := "testdata"
	expectedPaths := []string{"testdata/empty-crd.yaml", "testdata/invalid-crd.yml", "testdata/sample-crd.yaml"}

	// Test
	manifestPaths, err := getManifestPaths(manifestSrcDir)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, expectedPaths, manifestPaths)
}

func TestGetManifestPaths_EmptySrc(t *testing.T) {
	// Setup
	manifestSrcDir := ""

	// Test
	_, err := getManifestPaths(manifestSrcDir)

	// Verify
	require.ErrorContains(t, err, "manifest source directory not defined")
}

func TestGetManifestPaths_InvalidSrc(t *testing.T) {
	// Setup
	manifestSrcDir := "not-real"

	// Test
	_, err := getManifestPaths(manifestSrcDir)

	// Verify
	require.ErrorContains(t, err, "reading manifest source dir 'not-real': open not-real: no such file or directory")
}

func TestGetManifestPaths_NoManifests(t *testing.T) {
	// Setup
	require.NoError(t, os.Mkdir("downloaded-manifests", 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll("downloaded-manifests"))
	}()

	// Test
	manifestPaths, err := getManifestPaths("downloaded-manifests")

	// Verify
	require.NoError(t, err)
	assert.Nil(t, manifestPaths)
}

func TestManifestImages_InvalidURL(t *testing.T) {
	// Setup
	require.NoError(t, os.Mkdir("downloaded-manifests", 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll("downloaded-manifests"))
	}()

	manifestURLs := []string{
		"k8s.io/examples/application/nginx-app.yaml",
	}

	// Test
	_, err := ManifestImages(manifestURLs, "")

	// Verify
	require.ErrorContains(t, err, "downloading manifests: downloading manifest 'k8s.io/examples/application/nginx-app.yaml': executing request: Get \"k8s.io/examples/application/nginx-app.yaml\": unsupported protocol scheme \"\"")
}

func TestManifestImages_LocalManifestDirNotDefined(t *testing.T) {
	// Test
	containerImages, err := ManifestImages(nil, "")

	// Verify
	require.NoError(t, err)
	assert.Empty(t, containerImages)
}

func TestManifestImages_InvalidLocalManifestsDir(t *testing.T) {
	// Setup
	localManifestsDir := "does-not-exist"

	// Test
	_, err := ManifestImages(nil, localManifestsDir)

	// Verify
	require.ErrorContains(t, err, "getting local manifest paths: reading manifest source dir 'does-not-exist': open does-not-exist: no such file or directory")
}

func TestDownloadManifests_NoManifest(t *testing.T) {
	// Setup
	manifestDownloadDest := ""

	// Test
	manifestPaths, err := DownloadManifests(nil, manifestDownloadDest)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 0, len(manifestPaths))
}

func TestDownloadManifests_InvalidURL(t *testing.T) {
	// Setup
	manifestURLs := []string{"k8s.io/examples/application/nginx-app.yaml"}
	manifestDownloadDest := ""

	// Test
	manifestPaths, err := DownloadManifests(manifestURLs, manifestDownloadDest)

	// Verify
	require.ErrorContains(t, err, "downloading manifest 'k8s.io/examples/application/nginx-app.yaml': executing request: Get \"k8s.io/examples/application/nginx-app.yaml\": unsupported protocol scheme \"")
	assert.Equal(t, 0, len(manifestPaths))
}

func TestManifestImages_InvalidLocalManifest(t *testing.T) {
	// Setup
	require.NoError(t, os.Mkdir(localManifestsSrcDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(localManifestsSrcDir))
	}()

	localSampleManifestPath := filepath.Join("testdata", "invalid-crd.yml")
	err := fileio.CopyFile(localSampleManifestPath, filepath.Join(localManifestsSrcDir, "invalid-crd.yml"), fileio.NonExecutablePerms)
	require.NoError(t, err)

	// Test
	_, err = ManifestImages(nil, localManifestsSrcDir)

	// Verify
	require.ErrorContains(t, err, "reading manifest: error unmarshalling manifest yaml")
}
