//go:build integration

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
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes.Manifests.URLs = []string{
		"https://k8s.io/examples/application/nginx-app.yaml",
	}

	k8sCombDir := filepath.Join(ctx.CombustionDir, k8sDir)
	require.NoError(t, os.Mkdir(k8sCombDir, os.ModePerm))

	downloadedManifestsPath := filepath.Join(k8sDir, k8sManifestsDir)
	downloadedManifestsDestDir := filepath.Join(k8sCombDir, k8sManifestsDir)
	expectedDownloadedFilePath := filepath.Join(downloadedManifestsDestDir, "dl-manifest-1.yaml")

	// Test
	manifestsPath, err := configureManifests(ctx)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, downloadedManifestsPath, manifestsPath)

	assert.FileExists(t, expectedDownloadedFilePath)
	b, err := os.ReadFile(expectedDownloadedFilePath)
	require.NoError(t, err)

	contents := string(b)
	assert.Contains(t, contents, "apiVersion: v1")
	assert.Contains(t, contents, "name: my-nginx-svc")
	assert.Contains(t, contents, "type: LoadBalancer")
	assert.Contains(t, contents, "image: nginx:1.14.2")
}

func TestConfigureKubernetes_SuccessfulRKE2ServerWithManifests(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+rke2r1",
		Network: image.Network{
			APIVIP:  "192.168.122.100",
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
		},
	}
	ctx.KubernetesScriptDownloader = mockKubernetesScriptDownloader{
		downloadScript: func(distribution, destPath string) (string, error) {
			return "install-k8s.sh", nil
		},
	}
	ctx.KubernetesArtefactDownloader = mockKubernetesArtefactDownloader{
		downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, installPath, imagesPath string) error {
			return nil
		},
	}

	ctx.ImageDefinition.Kubernetes.Manifests.URLs = []string{
		"https://k8s.io/examples/application/nginx-app.yaml",
	}

	k8sCombDir := filepath.Join(ctx.CombustionDir, k8sDir)
	require.NoError(t, os.Mkdir(k8sCombDir, os.ModePerm))

	localManifestsSrcDir := filepath.Join(ctx.ImageConfigDir, k8sDir, k8sManifestsDir)
	require.NoError(t, os.MkdirAll(localManifestsSrcDir, 0o755))

	localSampleManifestPath := filepath.Join("..", "registry", "testdata", "sample-crd.yaml")
	err := fileio.CopyFile(localSampleManifestPath, filepath.Join(localManifestsSrcDir, "sample-crd.yaml"), fileio.NonExecutablePerms)
	require.NoError(t, err)

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
	assert.Contains(t, contents, "cp kubernetes/images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp server.yaml /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=kubernetes/install")
	assert.Contains(t, contents, "./install-k8s.sh")
	assert.Contains(t, contents, "systemctl enable rke2-server.service")
	assert.Contains(t, contents, "mkdir -p /var/lib/rancher/rke2/server/manifests/")
	assert.Contains(t, contents, "cp kubernetes/manifests/* /var/lib/rancher/rke2/server/manifests/")
	assert.Contains(t, contents, "cp "+registryMirrorsFileName+" /etc/rancher/rke2/registries.yaml")

	// Config file assertions
	configPath := filepath.Join(ctx.CombustionDir, "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	require.Contains(t, configContents, "cni")
	assert.Equal(t, "cilium", configContents["cni"], "default CNI is not set")
	assert.Equal(t, nil, configContents["server"])
	assert.Equal(t, []any{"192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Downloaded manifest assertions
	manifestPath := filepath.Join(k8sCombDir, k8sManifestsDir, "dl-manifest-1.yaml")
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

	// Local manifest assertions
	manifest := filepath.Join(k8sCombDir, k8sManifestsDir, "sample-crd.yaml")
	info, err = os.Stat(manifest)
	require.NoError(t, err)
	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(manifest)
	require.NoError(t, err)

	contents = string(b)
	assert.Contains(t, contents, "apiVersion: \"custom.example.com/v1\"")
	assert.Contains(t, contents, "app: complex-application")
	assert.Contains(t, contents, "- name: redis-container")
}
