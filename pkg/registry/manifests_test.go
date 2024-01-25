package registry

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

func TestReadManifest(t *testing.T) {
	// Setup
	manifestPath := filepath.Join("testdata", "sample-crd.yaml")

	// Test
	manifestData, err := readManifest(manifestPath)

	// Verify
	require.NoError(t, err)

	data, ok := manifestData.(map[string]any)
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
	require.ErrorContains(t, err, "error unmarshalling manifest yaml")
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
	var extractedImagesSet = make(map[string]string)
	manifestPath := filepath.Join("testdata", "sample-crd.yaml")
	manifestData, err := readManifest(manifestPath)
	require.NoError(t, err)

	expectedImages := []string{"nginx:latest", "node:14", "custom-api:1.2.3", "mysql:5.7", "redis:6.0"}
	sort.Strings(expectedImages)

	// Test
	storeManifestImageNames(manifestData, extractedImagesSet)
	allImages := make([]string, 0, len(extractedImagesSet))
	for uniqueImage := range extractedImagesSet {
		allImages = append(allImages, uniqueImage)
	}
	sort.Strings(allImages)

	// Verify
	assert.Equal(t, expectedImages, allImages)
}

func TestFindImagesInManifestEmptyManifest(t *testing.T) {
	// Setup
	var extractedImagesSet = make(map[string]string)
	var manifestData any

	// Test
	storeManifestImageNames(manifestData, extractedImagesSet)

	// Verify
	assert.Equal(t, map[string]string{}, extractedImagesSet)
}

func TestGetLocalManifestPaths(t *testing.T) {
	// Setup
	manifestSrcDir := "testdata"
	expectedPaths := []string{"testdata/empty-crd.yaml", "testdata/invalid-crd.yaml", "testdata/sample-crd.yaml"}

	// Test
	manifestPaths, err := getLocalManifestPaths(manifestSrcDir)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, expectedPaths, manifestPaths)
}

func TestGetLocalManifestPathsEmptySrc(t *testing.T) {
	// Setup
	manifestSrcDir := ""

	// Test
	_, err := getLocalManifestPaths(manifestSrcDir)

	// Verify
	require.ErrorContains(t, err, "manifest source directory not defined")
}

func TestGetLocalManifestPathsInvalidSrc(t *testing.T) {
	// Setup
	manifestSrcDir := "not-real"

	// Test
	_, err := getLocalManifestPaths(manifestSrcDir)

	// Verify
	require.ErrorContains(t, err, "reading manifest source dir 'not-real': open not-real: no such file or directory")
}

func TestGetLocalManifestPathsNoManifests(t *testing.T) {
	// Setup
	require.NoError(t, os.Mkdir("downloaded-manifests", 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll("downloaded-manifests"))
	}()

	// Test
	manifestPaths, err := getLocalManifestPaths("downloaded-manifests")

	// Verify
	require.NoError(t, err)
	assert.Nil(t, manifestPaths)
}

func TestGetAllImagesInvalidURL(t *testing.T) {
	// Setup
	require.NoError(t, os.Mkdir("downloaded-manifests", 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll("downloaded-manifests"))
	}()

	manifestURLs := []string{
		"k8s.io/examples/application/nginx-app.yaml",
	}

	// Test
	_, err := GetAllImages(nil, manifestURLs, "", "downloaded-manifests")

	// Verify
	require.ErrorContains(t, err, "error downloading manifests: downloading manifest 'k8s.io/examples/application/nginx-app.yaml': executing request: Get \"k8s.io/examples/application/nginx-app.yaml\": unsupported protocol scheme \"\"")
}

func TestGetAllImagesInvalidDownloadDestination(t *testing.T) {
	// Setup
	manifestURLs := []string{""}
	manifestDownloadDest := ""

	// Test
	_, err := GetAllImages(nil, manifestURLs, "", manifestDownloadDest)

	// Verify
	require.ErrorContains(t, err, "manifest download destination directory not defined")
}

func TestGetAllImagesLocalManifestDirNotDefined(t *testing.T) {
	// Test
	containerImages, err := GetAllImages(nil, nil, "", "")

	// Verify
	require.NoError(t, err)
	assert.Empty(t, containerImages)
}

func TestGetAllImagesInvalidLocalManifestsDir(t *testing.T) {
	// Setup
	localManifestsDir := "does-not-exist"

	// Test
	_, err := GetAllImages(nil, nil, localManifestsDir, "")

	// Verify
	require.ErrorContains(t, err, "error getting local manifest paths: reading manifest source dir 'does-not-exist': open does-not-exist: no such file or directory")
}

func TestDownloadManifestsNoManifest(t *testing.T) {
	// Setup
	manifestDownloadDest := ""

	// Test
	manifestPaths, err := downloadManifests(nil, manifestDownloadDest)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 0, len(manifestPaths))
}

func TestDownloadManifestsInvalidURL(t *testing.T) {
	// Setup
	manifestURLs := []string{"k8s.io/examples/application/nginx-app.yaml"}
	manifestDownloadDest := ""

	// Test
	manifestPaths, err := downloadManifests(manifestURLs, manifestDownloadDest)

	// Verify
	require.ErrorContains(t, err, "downloading manifest 'k8s.io/examples/application/nginx-app.yaml': executing request: Get \"k8s.io/examples/application/nginx-app.yaml\": unsupported protocol scheme \"")
	assert.Equal(t, 0, len(manifestPaths))
}

func TestGetAllImagesInvalidLocalManifest(t *testing.T) {
	// Setup
	localManifestSrcDir := "local-manifests"
	require.NoError(t, os.Mkdir(localManifestSrcDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(localManifestSrcDir))
	}()

	localSampleManifestPath := filepath.Join("testdata", "invalid-crd.yaml")
	err := fileio.CopyFile(localSampleManifestPath, filepath.Join(localManifestSrcDir, "invalid-crd.yaml"), fileio.NonExecutablePerms)
	require.NoError(t, err)

	// Test
	_, err = GetAllImages(nil, nil, localManifestSrcDir, "")

	// Verify
	require.ErrorContains(t, err, "error reading manifest error unmarshalling manifest yaml")
}
