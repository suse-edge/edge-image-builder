package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"gopkg.in/yaml.v3"
)

func TestConfigureManifestsValidDownload(t *testing.T) {
	// Setup
	manifestURLs := []string{
		"https://k8s.io/examples/application/nginx-app.yaml",
	}

	require.NoError(t, os.Mkdir(k8sCombustionDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(k8sCombustionDir))
	}()
	downloadedManifestsDestDir := filepath.Join(k8sCombustionDir, "manifests")
	expectedDownloadedFilePath := filepath.Join(downloadedManifestsDestDir, "dl-manifest-1.yaml")

	// Test
	err := configureManifests(k8sCombustionDir, false, "", manifestURLs)

	// Verify
	require.NoError(t, err)
	assert.FileExists(t, expectedDownloadedFilePath)

	b, err := os.ReadFile(expectedDownloadedFilePath)
	require.NoError(t, err)

	contents := string(b)
	assert.Contains(t, contents, "apiVersion: v1")
	assert.Contains(t, contents, "name: my-nginx-svc")
	assert.Contains(t, contents, "type: LoadBalancer")
	assert.Contains(t, contents, "image: nginx:1.14.2")
}

func TestConfigureKubernetes_SuccessfulRKE2ServerWithManifestURLs(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+rke2r1",
	}
	ctx.KubernetesScriptInstaller = mockKubernetesScriptInstaller{
		installScript: func(distribution, sourcePath, destPath string) error {
			return nil
		},
	}
	ctx.KubernetesArtefactDownloader = mockKubernetesArtefactDownloader{
		downloadArtefacts: func(arch image.Arch, version, cni string, multusEnabled bool, destPath string) (string, string, error) {
			return serverInstaller, serverImages, nil
		},
	}

	ctx.ImageDefinition.Kubernetes.Manifests.URLs = []string{
		"https://k8s.io/examples/application/nginx-app.yaml",
	}

	scripts, err := configureKubernetes(ctx)
	require.NoError(t, err)
	require.Len(t, scripts, 1)

	// Script file assertions
	scriptPath := filepath.Join(ctx.CombustionDir, scripts[0])

	info, err := os.Stat(scriptPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.ExecutablePerms, info.Mode())

	b, err := os.ReadFile(scriptPath)
	require.NoError(t, err)

	contents := string(b)
	assert.NotContains(t, contents, "export INSTALL_RKE2_TYPE=server",
		"INSTALL_RKE2_TYPE is set when the definition file does not explicitly set it")
	assert.Contains(t, contents, "cp server-images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp rke2_config.yaml /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=server-installer")
	assert.Contains(t, contents, "systemctl enable rke2-server.service")

	// Config file assertions
	configPath := filepath.Join(ctx.CombustionDir, "rke2_config.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	require.Contains(t, configContents, "cni")
	assert.Equal(t, "cilium", configContents["cni"], "default CNI is not set")

	// Downloaded manifest assertions
	manifestPath := filepath.Join(ctx.CombustionDir, "kubernetes", "manifests", "dl-manifest-1.yaml")
	info, err = os.Stat(manifestPath)
	require.NoError(t, err)
	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(manifestPath)
	require.NoError(t, err)

	contents = string(b)
	assert.Contains(t, contents, "apiVersion: v1")
	assert.Contains(t, contents, "name: my-nginx-svc")
	assert.Contains(t, contents, "type: LoadBalancer")
	assert.Contains(t, contents, "image: nginx:1.14.2")
}
