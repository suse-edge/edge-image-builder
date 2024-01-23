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
)

func TestWriteHaulerManifestValidManifest(t *testing.T) {
	// Setup
	haulerManifestYamlName := "hauler-manifest.yaml"
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
			ContainerImages: []image.ContainerImage{
				{
					Name: "hello-world:latest",
				},
				{
					Name:           "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
					SupplyChainKey: "carbide-key.pub",
				},
			},
			HelmCharts: []image.HelmChart{
				{
					Name:    "rancher",
					RepoURL: "https://releases.rancher.com/server-charts/stable",
					Version: "2.8.0",
				},
			},
		},
	}

	// Test
	err := writeHaulerManifest(ctx, ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages, ctx.ImageDefinition.EmbeddedArtifactRegistry.HelmCharts, haulerManifestYamlName)

	// Verify
	require.NoError(t, err)

	manifestFileName := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	_, err = os.Stat(manifestFileName)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(manifestFileName)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, "- name: hello-world:latest")
	assert.Contains(t, found, "- name: rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1")
	assert.Contains(t, found, "repoURL: https://releases.rancher.com/server-charts/stable")
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

	registryScriptPath := filepath.Join(ctx.CombustionDir, "13-embedded-registry.sh")
	_, err = os.Stat(registryScriptPath)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(registryScriptPath)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, "mv ./registry/* /opt/hauler/")
	assert.Contains(t, found, "mv hauler /usr/local/bin/hauler")
	assert.Contains(t, found, "systemctl enable eib-embedded-registry.service")
	assert.Contains(t, found, "  ExecStartPre=/bin/bash -c 'for file in /opt/hauler/*.tar.zst; do /usr/local/bin/hauler store load \\$file; done'\n")
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

func TestIsEmbeddedArtifactRegistryEmpty(t *testing.T) {
	tests := []struct {
		name     string
		registry image.EmbeddedArtifactRegistry
		isEmpty  bool
	}{
		{
			name: "Both Defined",
			registry: image.EmbeddedArtifactRegistry{
				HelmCharts: []image.HelmChart{
					{
						Name:    "rancher",
						RepoURL: "https://releases.rancher.com/server-charts/stable",
						Version: "2.8.0",
					},
				},
				ContainerImages: []image.ContainerImage{
					{
						Name:           "hello-world:latest",
						SupplyChainKey: "",
					},
				},
			},
			isEmpty: false,
		},
		{
			name: "Chart Defined",
			registry: image.EmbeddedArtifactRegistry{
				HelmCharts: []image.HelmChart{
					{
						Name:    "rancher",
						RepoURL: "https://releases.rancher.com/server-charts/stable",
						Version: "2.8.0",
					},
				},
			},
			isEmpty: false,
		},
		{
			name: "Image Defined",
			registry: image.EmbeddedArtifactRegistry{
				ContainerImages: []image.ContainerImage{
					{
						Name:           "hello-world:latest",
						SupplyChainKey: "",
					},
				},
			},
			isEmpty: false,
		},
		{
			name:    "None Defined",
			isEmpty: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := image.Context{
				ImageDefinition: &image.Definition{
					Kubernetes:               image.Kubernetes{HelmCharts: test.registry.HelmCharts},
					EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{ContainerImages: test.registry.ContainerImages},
				},
			}
			result := IsEmbeddedArtifactRegistryAndKubernetesManifestsEmpty(&ctx)
			assert.Equal(t, test.isEmpty, result)
		})
	}
}
