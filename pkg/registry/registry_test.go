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

func TestRegistry_New_InvalidManifestURL(t *testing.T) {
	buildDir := filepath.Join(os.TempDir(), "_build-registry-error")
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

	_, err := New(ctx, "", nil, "")
	require.Error(t, err)

	assert.ErrorContains(t, err, "downloading manifests: downloading manifest 'k8s.io/examples/application/nginx-app.yaml'")
	assert.ErrorContains(t, err, "unsupported protocol scheme")
}

func TestDownloadManifests_NoManifest(t *testing.T) {
	manifestPaths, err := downloadManifests(nil, "")

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

func TestRegistry_ContainerImages(t *testing.T) {
	manifestsDir := filepath.Join(os.TempDir(), "_manifests")
	require.NoError(t, os.MkdirAll(manifestsDir, os.ModePerm))
	defer func() {
		assert.NoError(t, os.RemoveAll(manifestsDir))
	}()

	assert.NoError(t, fileio.CopyFile("testdata/sample-crd.yaml", filepath.Join(manifestsDir, "sample-crd.yaml"), fileio.NonExecutablePerms))

	registry := Registry{
		embeddedImages: []image.ContainerImage{
			{
				Name: "hello-world",
			},
			{
				Name: "nginx:latest",
			},
		},
		manifestsDir: manifestsDir,
		helmCharts: []*helmChart{
			{
				HelmChart: image.HelmChart{
					Name: "apache",
				},
			},
		},
		helmClient: mockHelmClient{
			templateFunc: func(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error) {
				return []map[string]any{
					{
						"kind":  "Deployment",
						"image": "httpd",
					},
					{
						"kind": "Service",
					},
				}, nil
			},
		},
	}

	images, err := registry.ContainerImages()
	require.NoError(t, err)

	assert.ElementsMatch(t, images, []string{
		// embedded images
		"hello-world",
		"nginx:latest",
		// manifest images
		"node:14",
		"custom-api:1.2.3",
		"mysql:5.7",
		"redis:6.0",
		"nginx:1.14.2",
		// chart images
		"httpd",
	})
}

func TestDeduplicateContainerImages(t *testing.T) {
	embeddedImages := []image.ContainerImage{
		{
			Name: "hello-world:latest",
		},
		{
			Name: "embedded-image:1.0.0",
		},
	}

	manifestImages := []string{
		"hello-world:latest",
		"manifest-image:1.0.0",
	}

	chartImages := []string{
		"hello-world:latest",
		"chart-image:1.0.0",
		"chart-image:1.0.0",
		"chart-image:1.0.1",
		"chart-image:2.0.0",
	}

	assert.ElementsMatch(t, []string{
		"hello-world:latest",
		"embedded-image:1.0.0",
		"manifest-image:1.0.0",
		"chart-image:1.0.0",
		"chart-image:1.0.1",
		"chart-image:2.0.0",
	}, deduplicateContainerImages(embeddedImages, manifestImages, chartImages))
}
