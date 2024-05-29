//go:build integration

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

func TestDownloadManifests(t *testing.T) {
	// Setup
	manifestDownloadDest := "downloaded-manifests"
	require.NoError(t, os.Mkdir(manifestDownloadDest, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(manifestDownloadDest))
	}()

	expectedFilePath := filepath.Join(manifestDownloadDest, "dl-manifest-1.yaml")

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

func TestManifestImages(t *testing.T) {
	// Setup
	expectedContainerImages := []string{
		"custom-api:1.2.3",
		"mysql:5.7",
		"redis:6.0",
		"nginx:latest",
		"node:14",
		"nginx:1.14.2",
	}

	localManifestsDir := "local-manifests"

	require.NoError(t, os.Mkdir(localManifestsDir, 0o755))
	defer func() {
		assert.NoError(t, os.RemoveAll(localManifestsDir))
	}()

	sourceManifest := filepath.Join("testdata", "sample-crd.yaml")
	destinationManifest := filepath.Join(localManifestsDir, "sample-crd.yaml")

	require.NoError(t, fileio.CopyFile(sourceManifest, destinationManifest, fileio.NonExecutablePerms))

	buildDir := filepath.Join(os.TempDir(), "_manifests-integration")
	require.NoError(t, os.MkdirAll(buildDir, os.ModePerm))
	defer func() {
		assert.NoError(t, os.RemoveAll(buildDir))
	}()

	ctx := &image.Context{
		BuildDir: buildDir,
		ImageDefinition: &image.Definition{
			Kubernetes: image.Kubernetes{
				Manifests: image.Manifests{
					URLs: []string{"https://k8s.io/examples/application/nginx-app.yaml"},
				},
			},
		},
	}

	registry, err := New(ctx, nil, localManifestsDir)
	require.NoError(t, err)

	// Test
	containerImages, err := registry.ManifestImages()

	// Verify
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedContainerImages, containerImages)
}
