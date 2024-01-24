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

func TestDownloadManifests(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()
	destPath := ctx.CombustionDir
	expectedFilePath := filepath.Join(ctx.CombustionDir, "manifest-1.yaml")

	ctx.ImageDefinition.Kubernetes.Manifests.URLs = []string{
		"https://k8s.io/examples/application/nginx-app.yaml",
	}

	// Test
	manifestPaths, err := downloadManifests(ctx, destPath)

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

func TestDownloadManifestsNoManifest(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes.Manifests.URLs = []string{}

	// Test
	manifestPaths, err := downloadManifests(ctx, "")

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 0, len(manifestPaths))
}

func TestDownloadManifestsInvalidURL(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes.Manifests.URLs = []string{
		"k8s.io/examples/application/nginx-app.yaml",
	}

	// Test
	manifestPaths, err := downloadManifests(ctx, "")

	// Verify
	require.ErrorContains(t, err, "downloading manifest 'k8s.io/examples/application/nginx-app.yaml': executing request: Get \"k8s.io/examples/application/nginx-app.yaml\": unsupported protocol scheme \"")
	assert.Equal(t, 0, len(manifestPaths))
}

func TestGetAllImages(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	expectedContainerImages := []image.ContainerImage{
		{
			Name: "nginx:latest",
		},
		{
			Name: "node:14",
		},
		{
			Name: "mysql:5.7",
		},
		{
			Name: "redis:6.0",
		},
		{
			Name: "custom-api:1.2.3",
		},
		{
			Name:           "quay.io/podman/hello",
			SupplyChainKey: "sample-key",
		},
	}
	sort.Slice(expectedContainerImages, func(i, j int) bool {
		return expectedContainerImages[i].Name < expectedContainerImages[j].Name
	})

	localManifestSourceDir := filepath.Join(ctx.ImageConfigDir, "kubernetes", "manifests")
	err := os.MkdirAll(localManifestSourceDir, os.ModePerm)
	require.NoError(t, err)

	localSampleManifestPath := filepath.Join("testdata", "sample-crd.yaml")
	err = fileio.CopyFile(localSampleManifestPath, filepath.Join(localManifestSourceDir, "sample-crd.yaml"), fileio.NonExecutablePerms)
	require.NoError(t, err)

	ctx.ImageDefinition = &image.Definition{
		EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
			ContainerImages: []image.ContainerImage{
				{
					Name:           "quay.io/podman/hello",
					SupplyChainKey: "sample-key",
				},
			},
		},
		Kubernetes: image.Kubernetes{
			Manifests: image.Manifests{
				URLs: []string{
					"https://k8s.io/examples/application/nginx-app.yaml",
				},
			},
		},
	}

	// Test
	containerImages, err := GetAllImages(ctx)
	sort.Slice(containerImages, func(i, j int) bool {
		return containerImages[i].Name < containerImages[j].Name
	})

	// Verify
	require.NoError(t, err)
	assert.Equal(t, expectedContainerImages, containerImages)
}

func TestGetAllImagesInvalidLocalManifest(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	localManifestSourceDir := filepath.Join(ctx.ImageConfigDir, "kubernetes", "manifests")
	err := os.MkdirAll(localManifestSourceDir, os.ModePerm)
	require.NoError(t, err)

	localSampleManifestPath := filepath.Join("testdata", "invalid-crd.yaml")
	err = fileio.CopyFile(localSampleManifestPath, filepath.Join(localManifestSourceDir, "sample-crd.yaml"), fileio.NonExecutablePerms)
	require.NoError(t, err)

	// Test
	_, err = GetAllImages(ctx)

	// Verify
	require.ErrorContains(t, err, "error reading manifest error unmarshalling manifest yaml")
}

func TestGetAllImagesInvalidURL(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes.Manifests.URLs = []string{
		"k8s.io/examples/application/nginx-app.yaml",
	}

	// Test
	_, err := GetAllImages(ctx)

	// Verify
	require.ErrorContains(t, err, "error downloading manifests: downloading manifest")
}

func TestGetAllImagesLocalManifestDirNotDefined(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	_, err := GetAllImages(ctx)

	// Verify
	require.ErrorContains(t, err, "error getting local manifest paths:")
}
