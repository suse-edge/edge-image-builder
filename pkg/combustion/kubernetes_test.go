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
	downloadRKE2Artefacts func(arch image.Arch, version, cni string, multusEnabled bool, installPath, imagesPath string) error
	downloadK3sArtefacts  func(arch image.Arch, version, installPath, imagesPath string) error
}

func (m mockKubernetesArtefactDownloader) DownloadRKE2Artefacts(
	arch image.Arch,
	version string,
	cni string,
	multusEnabled bool,
	installPath string,
	imagesPath string,
) error {
	if m.downloadRKE2Artefacts != nil {
		return m.downloadRKE2Artefacts(arch, version, cni, multusEnabled, installPath, imagesPath)
	}

	panic("not implemented")
}

func (m mockKubernetesArtefactDownloader) DownloadK3sArtefacts(arch image.Arch, version, installPath, imagesPath string) error {
	if m.downloadK3sArtefacts != nil {
		return m.downloadK3sArtefacts(arch, version, installPath, imagesPath)
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

func TestConfigureKubernetes_ScriptInstallerErrorK3s(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+k3s1",
	}
	ctx.KubernetesScriptInstaller = mockKubernetesScriptInstaller{
		installScript: func(distribution, sourcePath, destPath string) error {
			return fmt.Errorf("some error")
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: copying k3s installer script: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_ScriptInstallerErrorRKE2(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+rke2r1",
	}
	ctx.KubernetesScriptInstaller = mockKubernetesScriptInstaller{
		installScript: func(distribution, sourcePath, destPath string) error {
			return fmt.Errorf("some error")
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: copying RKE2 installer script: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_ArtefactDownloaderErrorK3s(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+k3s1",
	}
	ctx.KubernetesScriptInstaller = mockKubernetesScriptInstaller{
		installScript: func(distribution, sourcePath, destPath string) error {
			return nil
		},
	}
	ctx.KubernetesArtefactDownloader = mockKubernetesArtefactDownloader{
		downloadK3sArtefacts: func(arch image.Arch, version string, installPath, imagesPath string) error {
			return fmt.Errorf("some error")
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: downloading k3s artefacts: downloading artefacts: some error")
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
		downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, installPath, imagesPath string) error {
			return fmt.Errorf("some error")
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: downloading RKE2 artefacts: downloading artefacts: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_SuccessfulSingleNodeK3sCluster(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+k3s1",
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
		downloadK3sArtefacts: func(arch image.Arch, version string, installPath, imagesPath string) error {
			binary := filepath.Join(installPath, "cool-k3s-binary")
			return os.WriteFile(binary, nil, os.ModePerm)
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
	assert.Contains(t, contents, "cp kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp server.yaml /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "cp k8s-vip.yaml /var/lib/rancher/k3s/server/manifests/k8s-vip.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/k3s/bin")
	assert.Contains(t, contents, "chmod +x kubernetes/install/cool-k3s-binary")
	assert.Contains(t, contents, "cp kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")

	// Config file assertions
	configPath := filepath.Join(ctx.CombustionDir, "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Nil(t, configContents["cni"])
	assert.Nil(t, configContents["server"])
	assert.Equal(t, []any{"192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
}

func TestConfigureKubernetes_SuccessfulMultiNodeK3sCluster(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.29.0+k3s1",
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
		downloadK3sArtefacts: func(arch image.Arch, version, installPath, imagesPath string) error {
			binary := filepath.Join(installPath, "cool-k3s-binary")
			return os.WriteFile(binary, nil, os.ModePerm)
		},
	}

	serverConfig := map[string]any{
		"token": "123",
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
	assert.Contains(t, contents, "cp kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "cp k8s-vip.yaml /var/lib/rancher/k3s/server/manifests/k8s-vip.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_EXEC=$NODETYPE")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/k3s/bin")
	assert.Contains(t, contents, "chmod +x kubernetes/install/cool-k3s-binary")
	assert.Contains(t, contents, "cp kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")

	// Server config file assertions
	configPath := filepath.Join(ctx.CombustionDir, "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://192.168.122.100:6443", configContents["server"])
	assert.Equal(t, []any{"192-168-122-100.sslip.io", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Nil(t, configContents["cluster-init"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.CombustionDir, "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.Equal(t, []any{"192-168-122-100.sslip.io", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Equal(t, true, configContents["cluster-init"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.CombustionDir, "agent.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://192.168.122.100:6443", configContents["server"])
	assert.Nil(t, configContents["tls-san"])
	assert.Nil(t, configContents["disable"])
	assert.Nil(t, configContents["cluster-init"])
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
		downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, installPath, imagesPath string) error {
			return nil
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
	assert.Contains(t, contents, "cp kubernetes/images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp server.yaml /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "cp k8s-vip.yaml /var/lib/rancher/rke2/server/manifests/k8s-vip.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=kubernetes/install")
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
		downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, installPath, imagesPath string) error {
			return nil
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
	assert.Contains(t, contents, "cp kubernetes/images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "cp k8s-vip.yaml /var/lib/rancher/rke2/server/manifests/k8s-vip.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=kubernetes/install")
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
	assert.Nil(t, configContents["tls-san"])
}

func TestConfigureKubernetes_InvalidManifestURL(t *testing.T) {
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
		downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, installpath, imagesPath string) error {
			return nil
		},
	}
	ctx.ImageDefinition.Kubernetes.Manifests.URLs = []string{
		"k8s.io/examples/application/nginx-app.yaml",
	}
	k8sCombDir := filepath.Join(ctx.CombustionDir, k8sDir)
	require.NoError(t, os.Mkdir(k8sCombDir, os.ModePerm))

	_, err := configureKubernetes(ctx)

	require.ErrorContains(t, err, "configuring kubernetes manifests: downloading manifests to combustion dir: downloading manifest 'k8s.io/examples/application/nginx-app.yaml': executing request: Get \"k8s.io/examples/application/nginx-app.yaml\": unsupported protocol scheme \"\"")
}

func TestConfigureManifestsNoSetup(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	manifestsPath, err := configureManifests(ctx)

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "", manifestsPath)
}
