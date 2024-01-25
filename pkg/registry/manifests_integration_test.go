//go:build integration

package registry

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestDownloadManifests(t *testing.T) {
	// Setup
	manifestDownloadDest := "downloaded-manifests"
	require.NoError(t, os.Mkdir(manifestDownloadDest, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(manifestDownloadDest))
	}()

	expectedFilePath := filepath.Join(manifestDownloadDest, "manifest-1.yaml")

	manifestURLs := []string{
		"https://k8s.io/examples/application/nginx-app.yaml",
	}

	// Test
	manifestPaths, err := downloadManifests(manifestURLs, manifestDownloadDest)

	// Verify
	require.NoError(t, err)
	assert.FileExists(t, expectedFilePath)
	assert.Contains(t, manifestPaths, expectedFilePath)

	foundBytes, err := os.ReadFile(expectedFilePath)
	require.NoError(t, err)
	found := string(foundBytes)

	assert.Contains(t, found, "apiVersion: v1")
	assert.Contains(t, found, "image: nginx:1.14.2")
}

func TestGetAllImages(t *testing.T) {
	// Setup
	expectedContainerImages := []image.ContainerImage{
		{
			Name: "custom-api:1.2.3",
		},
		{
			Name:           "quay.io/podman/hello",
			SupplyChainKey: "sample-key",
		},
		{
			Name: "mysql:5.7",
		},
		{
			Name: "redis:6.0",
		},
		{
			Name: "nginx:latest",
		},
		{
			Name: "node:14",
		},
	}
	sort.Slice(expectedContainerImages, func(i, j int) bool {
		return expectedContainerImages[i].Name < expectedContainerImages[j].Name
	})

	manifestDownloadDest := "downloaded-manifests"
	require.NoError(t, os.Mkdir(manifestDownloadDest, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(manifestDownloadDest))
	}()

	localManifestSrcDir := "local-manifests"
	require.NoError(t, os.Mkdir(localManifestSrcDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(localManifestSrcDir))
	}()

	localSampleManifestPath := filepath.Join("testdata", "sample-crd.yaml")
	err := fileio.CopyFile(localSampleManifestPath, filepath.Join(localManifestSrcDir, "sample-crd.yaml"), fileio.NonExecutablePerms)
	require.NoError(t, err)

	embeddedContainerImages := []image.ContainerImage{
		{
			Name:           "quay.io/podman/hello",
			SupplyChainKey: "sample-key",
		},
	}
	manifestURLs := []string{"https://k8s.io/examples/application/nginx-app.yaml"}

	// Test
	containerImages, err := GetAllImages(embeddedContainerImages, manifestURLs, localManifestSrcDir, manifestDownloadDest)

	// Verify
	require.NoError(t, err)
	sort.Slice(containerImages, func(i, j int) bool {
		return containerImages[i].Name < containerImages[j].Name
	})
	assert.Equal(t, expectedContainerImages, containerImages)
}
