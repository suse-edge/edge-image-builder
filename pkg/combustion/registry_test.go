package combustion

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/registry"
)

func TestWriteHaulerManifestValidManifest(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	images := []string{
		"hello-world:latest",
		"ghcr.io/fluxcd/flux-cli@sha256:02aa820c3a9c57d67208afcfc4bce9661658c17d15940aea369da259d2b976dd",
	}

	// Test
	err := writeHaulerManifest(ctx, images)

	// Verify
	require.NoError(t, err)

	manifestFileName := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	_, err = os.Stat(manifestFileName)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(manifestFileName)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, "- name: hello-world:latest")
	assert.Contains(t, found, "- name: ghcr.io/fluxcd/flux-cli@sha256:02aa820c3a9c57d67208afcfc4bce9661658c17d15940aea369da259d2b976dd")
}

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
	_, err = os.Stat(registryScriptPath)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(registryScriptPath)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, registryDir)
	assert.Contains(t, found, registryPort)
	assert.Contains(t, found, registryTarName)
	assert.Contains(t, found, "mv hauler /opt/hauler/hauler")
	assert.Contains(t, found, "systemctl enable eib-embedded-registry.service")
	assert.Contains(t, found, "ExecStartPre=/opt/hauler/hauler store load")
}

func TestCopyHaulerBinary(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	haulerBinaryPath := filepath.Join(ctx.BuildDir, fmt.Sprintf("hauler-%s", string(ctx.ImageDefinition.Image.Arch)))
	err := os.WriteFile(haulerBinaryPath, []byte(""), fileio.ExecutablePerms)
	require.NoError(t, err)

	// Test
	err = copyHaulerBinary(ctx, haulerBinaryPath)

	// Verify
	require.NoError(t, err)

	haulerPath := filepath.Join(ctx.CombustionDir, "hauler")
	_, err = os.Stat(haulerPath)
	require.NoError(t, err)
}

func TestCopyHaulerBinaryNoFile(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	err := copyHaulerBinary(ctx, "")

	// Verify
	require.ErrorContains(t, err, "no such file")
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

	manifestFileName := filepath.Join(ctx.CombustionDir, registryMirrorsFileName)
	_, err = os.Stat(manifestFileName)
	require.NoError(t, err)

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

	charts := []*registry.HelmChart{
		{
			Filename: "metallb.yaml",
			Resources: []map[string]any{
				{
					"apiVersion": "helm.cattle.io/v1",
					"kind":       "HelmChart",
					"metadata": map[string]any{
						"name":      "metallb",
						"namespace": "metallb-system",
					},
					"spec": map[string]any{
						"chart":           "metallb",
						"repo":            "https://suse-edge.github.io/charts",
						"targetNamespace": "metallb-system",
					},
				},
				{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]any{
						"name": "metallb-system",
					},
					"spec": map[string]any{},
				},
			},
		},
		{
			Filename: "endpoint-copier-operator.yaml",
			Resources: []map[string]any{
				{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]any{
						"name": "endpoint-copier-operator",
					},
					"spec": map[string]any{},
				},
				{
					"apiVersion": "helm.cattle.io/v1",
					"kind":       "HelmChart",
					"metadata": map[string]any{
						"name":      "endpoint-copier-operator",
						"namespace": "endpoint-copier-operator",
					},
					"spec": map[string]any{
						"chart":           "endpoint-copier-operator",
						"repo":            "https://suse-edge.github.io/charts",
						"targetNamespace": "endpoint-copier-operator",
					},
				},
			},
		},
	}

	require.NoError(t, storeHelmCharts(ctx, charts))

	metalLBPath := filepath.Join(ctx.CombustionDir, k8sDir, k8sManifestsDir, "metallb.yaml")
	metalLBContents := `---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
    name: metallb
    namespace: metallb-system
spec:
    chart: metallb
    repo: https://suse-edge.github.io/charts
    targetNamespace: metallb-system
---
apiVersion: v1
kind: Namespace
metadata:
    name: metallb-system
spec: {}
`
	contents, err := os.ReadFile(metalLBPath)
	require.NoError(t, err)

	assert.Equal(t, metalLBContents, string(contents))

	endpointCopierOperatorPath := filepath.Join(ctx.CombustionDir, k8sDir, k8sManifestsDir, "endpoint-copier-operator.yaml")
	endpointCopierOperatorContents := `---
apiVersion: v1
kind: Namespace
metadata:
    name: endpoint-copier-operator
spec: {}
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
    name: endpoint-copier-operator
    namespace: endpoint-copier-operator
spec:
    chart: endpoint-copier-operator
    repo: https://suse-edge.github.io/charts
    targetNamespace: endpoint-copier-operator
`
	contents, err = os.ReadFile(endpointCopierOperatorPath)
	require.NoError(t, err)

	assert.Equal(t, endpointCopierOperatorContents, string(contents))
}
