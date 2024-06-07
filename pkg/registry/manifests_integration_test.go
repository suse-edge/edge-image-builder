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

func TestManifestImages(t *testing.T) {
	// Setup
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

	registry, err := New(ctx, localManifestsDir, nil, "")
	require.NoError(t, err)

	// Test
	containerImages, err := registry.manifestImages()

	// Verify
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"custom-api:1.2.3",
		"mysql:5.7",
		"redis:6.0",
		"nginx:latest",
		"node:14",
		"nginx:1.14.2",
	}, containerImages)
}
