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
	"gopkg.in/yaml.v3"
)

type mockKubernetesScriptInstaller struct {
	installScript func(distribution, sourcePath, destPath string) error
}

func (m mockKubernetesScriptInstaller) InstallScript(distribution, sourcePath, destPath string) error {
	if m.installScript != nil {
		return m.installScript(distribution, sourcePath, destPath)
	}

	panic("not implemented")
}

type mockKubernetesArtefactDownloader struct {
	downloadArtefacts func(arch image.Arch, version, cni string, multusEnabled bool, destPath string) (string, string, error)
}

func (m mockKubernetesArtefactDownloader) DownloadArtefacts(
	arch image.Arch,
	version string,
	cni string,
	multusEnabled bool,
	destPath string,
) (installPath string, imagesPath string, err error) {
	if m.downloadArtefacts != nil {
		return m.downloadArtefacts(arch, version, cni, multusEnabled, destPath)
	}

	panic("not implemented")
}

func TestConfigureKubernetes_Skipped(t *testing.T) {
	ctx := &image.Context{
		ImageDefinition: &image.Definition{},
	}

	scripts, err := configureKubernetes(ctx)
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_UnsupportedVersion(t *testing.T) {
	ctx := &image.Context{
		ImageDefinition: &image.Definition{
			Kubernetes: image.Kubernetes{
				Version: "v1.29.0",
			},
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "cannot configure kubernetes version: v1.29.0")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_UnimplementedK3S(t *testing.T) {
	ctx := &image.Context{
		ImageDefinition: &image.Definition{
			Kubernetes: image.Kubernetes{
				Version: "v1.29.0+k3s1",
			},
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: not implemented yet")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_ScriptInstallerErrorRKE2(t *testing.T) {
	ctx := &image.Context{
		ImageDefinition: &image.Definition{
			Kubernetes: image.Kubernetes{
				Version: "v1.29.0+rke2r1",
			},
		},
		KubernetesScriptInstaller: mockKubernetesScriptInstaller{
			installScript: func(distribution, sourcePath, destPath string) error {
				return fmt.Errorf("some error")
			},
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: copying RKE2 installer script: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_ArtefactDownloaderErrorRKE2(t *testing.T) {
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
			return "", "", fmt.Errorf("some error")
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: downloading RKE2 artefacts: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_SuccessfulSingleNodeRKE2Cluster(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+rke2r1",
		Network: image.Network{
			APIVIP:  "192.168.122.100",
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
		},
	}
	ctx.KubernetesScriptInstaller = mockKubernetesScriptInstaller{
		installScript: func(distribution, sourcePath, destPath string) error {
			return nil
		},
	}
	ctx.KubernetesArtefactDownloader = mockKubernetesArtefactDownloader{
		downloadArtefacts: func(arch image.Arch, version, cni string, multusEnabled bool, destPath string) (string, string, error) {
			return "server-installer", "server-images", nil
		},
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
	assert.Contains(t, contents, "cp server-images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp server.yaml /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "cp rke2-vip.yaml /var/lib/rancher/rke2/server/manifests/rke2-vip.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=server-installer")
	assert.Contains(t, contents, "systemctl enable rke2-server.service")

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
}

func TestConfigureKubernetes_SuccessfulMultiNodeRKE2Cluster(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+rke2r1",
		Network: image.Network{
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
			APIVIP:  "192.168.122.100",
		},
		Nodes: []image.Node{
			{
				Hostname: "node1.suse.com",
				Type:     "server",
			},
			{
				Hostname: "node2.suse.com",
				Type:     "agent",
			},
		},
	}
	ctx.KubernetesScriptInstaller = mockKubernetesScriptInstaller{
		installScript: func(distribution, sourcePath, destPath string) error {
			return nil
		},
	}
	ctx.KubernetesArtefactDownloader = mockKubernetesArtefactDownloader{
		downloadArtefacts: func(arch image.Arch, version, cni string, multusEnabled bool, destPath string) (string, string, error) {
			return "server-installer", "server-images", nil
		},
	}

	serverConfig := map[string]any{
		"token": "123",
		"cni":   "canal",
		"tls-san": []string{
			"192-168-122-100.sslip.io",
		},
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configDir := filepath.Join(ctx.ImageConfigDir, k8sDir, k8sConfigDir)
	require.NoError(t, os.MkdirAll(configDir, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "server.yaml"), b, os.ModePerm))

	scripts, err := configureKubernetes(ctx)
	require.NoError(t, err)
	require.Len(t, scripts, 1)

	// Script file assertions
	scriptPath := filepath.Join(ctx.CombustionDir, scripts[0])

	info, err := os.Stat(scriptPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.ExecutablePerms, info.Mode())

	b, err = os.ReadFile(scriptPath)
	require.NoError(t, err)

	contents := string(b)
	assert.Contains(t, contents, "hosts[node1.suse.com]=server")
	assert.Contains(t, contents, "hosts[node2.suse.com]=agent")
	assert.Contains(t, contents, "cp server-images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "cp rke2-vip.yaml /var/lib/rancher/rke2/server/manifests/rke2-vip.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=server-installer")
	assert.Contains(t, contents, "systemctl enable rke2-$NODETYPE.service")

	// Server config file assertions
	configPath := filepath.Join(ctx.CombustionDir, "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://192.168.122.100:9345", configContents["server"])
	assert.Equal(t, []any{"192-168-122-100.sslip.io", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.CombustionDir, "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.Equal(t, []any{"192-168-122-100.sslip.io", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.CombustionDir, "agent.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://192.168.122.100:9345", configContents["server"])
	assert.Equal(t, []any{"192-168-122-100.sslip.io", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
}
