package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/registry"
)

func TestCreateRegistryCommand(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	cmd, logFile, err := createRegistryCommand(ctx, "testName", []string{"--flag", "test"})

	// Verify
	require.NoError(t, err)
	require.NotNil(t, cmd)

	expectedCommand := "testName"
	expectedArgs := []string{"testName", "--flag", "test"}

	assert.Equal(t, expectedCommand, cmd.Path)
	assert.Equal(t, expectedArgs, cmd.Args)

	assert.Equal(t, logFile, cmd.Stdout)
	assert.Equal(t, logFile, cmd.Stderr)

	foundFile := filepath.Join(ctx.BuildDir, "embedded-registry.log")
	_, err = os.ReadFile(foundFile)
	require.NoError(t, err)
}

func TestWriteRegistryScript(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	_, err := writeRegistryScript(ctx)

	// Verify
	require.NoError(t, err)

	registryScriptPath := filepath.Join(ctx.CombustionDir, registryScriptName)

	foundBytes, err := os.ReadFile(registryScriptPath)
	require.NoError(t, err)

	found := string(foundBytes)
	assert.Contains(t, found, "cp $ARTEFACTS_DIR/registry/hauler /opt/hauler/hauler")
	assert.Contains(t, found, "cp $ARTEFACTS_DIR/registry/*-registry.tar.zst /opt/hauler/")
	assert.Contains(t, found, "systemctl enable eib-embedded-registry.service")
	assert.Contains(t, found, "ExecStartPre=/bin/sh -c '/opt/hauler/hauler store load *-registry.tar.zst'")
	assert.Contains(t, found, "ExecStart=/opt/hauler/hauler store serve registry -p 6545")
}

func TestIsEmbeddedArtifactRegistryConfigured(t *testing.T) {
	tests := []struct {
		name         string
		ctx          *image.Context
		isConfigured bool
	}{
		{
			name: "Everything Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
						ContainerImages: []image.ContainerImage{
							{
								Name: "nginx",
							},
						},
					},
					Kubernetes: image.Kubernetes{
						Manifests: image.Manifests{
							URLs: []string{
								"https://k8s.io/examples/application/nginx-app.yaml",
							},
						},
						Helm: image.Helm{
							Charts: []image.HelmChart{
								{
									Name:           "apache",
									RepositoryName: "apache-repo",
									Version:        "10.7.0",
								},
							},
						},
					},
				},
			},
			isConfigured: true,
		},
		{
			name: "Image Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
						ContainerImages: []image.ContainerImage{
							{
								Name: "nginx",
							},
						},
					},
				},
			},
			isConfigured: true,
		},
		{
			name: "Manifest URL Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					Kubernetes: image.Kubernetes{
						Manifests: image.Manifests{
							URLs: []string{
								"https://k8s.io/examples/application/nginx-app.yaml",
							},
						},
					},
				},
			},
			isConfigured: true,
		},
		{
			name: "Helm Charts Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					Kubernetes: image.Kubernetes{
						Helm: image.Helm{
							Charts: []image.HelmChart{
								{
									Name:           "apache",
									RepositoryName: "apache-repo",
									Version:        "10.7.0",
								},
							},
						},
					},
				},
			},
			isConfigured: true,
		},
		{
			name: "None Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{},
					Kubernetes:               image.Kubernetes{},
				},
			},
			isConfigured: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsEmbeddedArtifactRegistryConfigured(test.ctx)
			assert.Equal(t, test.isConfigured, result)
		})
	}
}

func TestWriteRegistryMirrorsValid(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	hostnames := []string{"hello-world:latest", "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1", "quay.io"}

	// Test
	err := writeRegistryMirrors(ctx, hostnames)

	// Verify
	require.NoError(t, err)

	manifestFileName := filepath.Join(ctx.ArtefactsDir, K8sDir, registryMirrorsFileName)

	foundBytes, err := os.ReadFile(manifestFileName)
	require.NoError(t, err)

	found := string(foundBytes)
	assert.Contains(t, found, "- \"http://localhost:6545\"")
	assert.Contains(t, found, "docker.io")
	assert.Contains(t, found, "rgcrprod.azurecr.us")
	assert.Contains(t, found, "quay.io")
}

func TestGetImageHostnames(t *testing.T) {
	// Setup
	images := []string{
		"hello-world:latest",
		"quay.io/podman/hello",
		"rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
	}
	expectedHostnames := []string{"quay.io", "rgcrprod.azurecr.us"}

	// Test
	hostnames := getImageHostnames(images)

	// Verify
	assert.Equal(t, expectedHostnames, hostnames)
}

func TestContainerImages(t *testing.T) {
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

	helmCharts := []*registry.HelmChart{
		{
			ContainerImages: []string{
				"hello-world:latest",
				"helm-image:1.0.0",
			},
		},
		{
			ContainerImages: []string{
				"helm-image:2.0.0",
			},
		},
	}

	assert.ElementsMatch(t, []string{
		"hello-world:latest",
		"embedded-image:1.0.0",
		"manifest-image:1.0.0",
		"helm-image:1.0.0",
		"helm-image:2.0.0",
	}, containerImages(embeddedImages, manifestImages, helmCharts))
}

func TestStoreHelmCharts(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	helmChart := &image.HelmChart{
		Name:                  "apache",
		RepositoryName:        "apache-repo",
		TargetNamespace:       "web",
		CreateNamespace:       true,
		InstallationNamespace: "kube-system",
		Version:               "10.7.0",
		ValuesFile:            "",
	}

	charts := []*registry.HelmChart{
		{
			CRD: registry.NewHelmCRD(helmChart, "some-content", `
values: content`, "oci://registry-1.docker.io/bitnamicharts"),
		},
	}

	require.NoError(t, storeHelmCharts(ctx, charts))

	apachePath := filepath.Join(ctx.ArtefactsDir, K8sDir, k8sManifestsDir, "apache.yaml")
	apacheContent := `apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
    name: apache
    namespace: kube-system
    annotations:
        edge.suse.com/repository-url: oci://registry-1.docker.io/bitnamicharts
        edge.suse.com/source: edge-image-builder
spec:
    version: 10.7.0
    valuesContent: |4-
        values: content
    chartContent: some-content
    targetNamespace: web
    createNamespace: true
`
	contents, err := os.ReadFile(apachePath)
	require.NoError(t, err)

	assert.Equal(t, apacheContent, string(contents))
}
