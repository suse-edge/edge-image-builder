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
	"gopkg.in/yaml.v3"
)

const kubernetesScriptInstaller = "install-kubernetes.sh"

type mockKubernetesScriptDownloader struct {
	downloadScript func(distribution, destPath string) (string, error)
}

func (m mockKubernetesScriptDownloader) DownloadInstallScript(distribution, destPath string) (string, error) {
	if m.downloadScript != nil {
		return m.downloadScript(distribution, destPath)
	}

	panic("not implemented")
}

type mockKubernetesArtefactDownloader struct {
	downloadRKE2Artefacts func(arch image.Arch, version, cni string, multusEnabled bool, ingressController string, installPath, imagesPath string) error
	downloadK3sArtefacts  func(arch image.Arch, version, installPath, imagesPath string) error
}

func (m mockKubernetesArtefactDownloader) DownloadRKE2Artefacts(
	arch image.Arch,
	version string,
	cni string,
	multusEnabled bool,
	ingressController string,
	installPath string,
	imagesPath string,
) error {
	if m.downloadRKE2Artefacts != nil {
		return m.downloadRKE2Artefacts(arch, version, cni, multusEnabled, ingressController, installPath, imagesPath)
	}

	panic("not implemented")
}

func (m mockKubernetesArtefactDownloader) DownloadK3sArtefacts(arch image.Arch, version, installPath, imagesPath string) error {
	if m.downloadK3sArtefacts != nil {
		return m.downloadK3sArtefacts(arch, version, installPath, imagesPath)
	}

	panic("not implemented")
}

type mockEmbeddedRegistry struct {
	helmChartsFunc      func() ([]*registry.HelmCRD, error)
	containerImagesFunc func() ([]string, error)
	manifestsPathFunc   func() string
}

func (m mockEmbeddedRegistry) HelmCharts() ([]*registry.HelmCRD, error) {
	if m.helmChartsFunc != nil {
		return m.helmChartsFunc()
	}

	panic("not implemented")
}

func (m mockEmbeddedRegistry) ContainerImages() ([]string, error) {
	if m.containerImagesFunc != nil {
		return m.containerImagesFunc()
	}

	panic("not implemented")
}

func (m mockEmbeddedRegistry) ManifestsPath() string {
	if m.manifestsPathFunc != nil {
		return m.manifestsPathFunc()
	}

	panic("not implemented")
}

func TestConfigureKubernetes_Skipped(t *testing.T) {
	ctx := &image.Context{
		ImageDefinition: &image.Definition{},
	}

	var c Combustion

	scripts, err := c.configureKubernetes(ctx)
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_UnsupportedVersion(t *testing.T) {
	ctx := &image.Context{
		ImageDefinition: &image.Definition{
			Kubernetes: image.Kubernetes{
				Version: "v1.30.3",
			},
		},
	}

	var c Combustion

	scripts, err := c.configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "cannot configure kubernetes version: v1.30.3")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_ScriptInstallerErrorK3s(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return "", fmt.Errorf("some error")
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: downloading k3s install script: downloading install script: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_ScriptInstallerErrorRKE2(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return "", fmt.Errorf("some error")
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: downloading RKE2 install script: downloading install script: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_ArtefactDownloaderErrorK3s(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadK3sArtefacts: func(arch image.Arch, version string, installPath, imagesPath string) error {
				return fmt.Errorf("some error")
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: downloading k3s artefacts: downloading artefacts: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_ArtefactDownloaderErrorRKE2(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, ingressController string, installPath, imagesPath string) error {
				return fmt.Errorf("some error")
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: downloading RKE2 artefacts: downloading artefacts: some error")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_Successful_SingleNode_K3s_IPv4(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIVIP4: "192.168.122.100",
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
		},
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadK3sArtefacts: func(arch image.Arch, version string, installPath, imagesPath string) error {
				binary := filepath.Join(installPath, "cool-k3s-binary")
				return os.WriteFile(binary, nil, os.ModePerm)
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/server.yaml /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/bin")
	assert.Contains(t, contents, "chmod +x $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.NotContains(t, contents, "sh set-node-ip.sh")

	// Config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

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

func TestConfigureKubernetes_Successful_SingleNode_K3s_IPv6(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIVIP6: "fd12:3456:789a::21",
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
		},
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadK3sArtefacts: func(arch image.Arch, version string, installPath, imagesPath string) error {
				binary := filepath.Join(installPath, "cool-k3s-binary")
				return os.WriteFile(binary, nil, os.ModePerm)
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/server.yaml /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "echo \"fd12:3456:789a::21 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/bin")
	assert.Contains(t, contents, "chmod +x $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.Contains(t, contents, "sh set-node-ip.sh")

	// Config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Nil(t, configContents["cni"])
	assert.Nil(t, configContents["server"])
	assert.Equal(t, []any{"fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
}

func TestConfigureKubernetes_Successful_SingleNode_K3s_Dualstack_PrioIPv4(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIVIP4: "192.168.122.100",
			APIVIP6: "fd12:3456:789a::21",
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
		},
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadK3sArtefacts: func(arch image.Arch, version string, installPath, imagesPath string) error {
				binary := filepath.Join(installPath, "cool-k3s-binary")
				return os.WriteFile(binary, nil, os.ModePerm)
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/server.yaml /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "echo \"fd12:3456:789a::21 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/bin")
	assert.Contains(t, contents, "chmod +x $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.Contains(t, contents, "sh set-node-ip.sh")

	// Config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Nil(t, configContents["cni"])
	assert.Nil(t, configContents["server"])
	assert.ElementsMatch(t, []any{"192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
}

func TestConfigureKubernetes_Successful_MultiNode_K3s_IPv4(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
			APIVIP4: "192.168.122.100",
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

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadK3sArtefacts: func(arch image.Arch, version, installPath, imagesPath string) error {
				binary := filepath.Join(installPath, "cool-k3s-binary")
				return os.WriteFile(binary, nil, os.ModePerm)
			},
		},
	}

	serverConfig := map[string]any{
		"token": "123",
		"tls-san": []string{
			"k8s-host.com",
		},
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configDir := filepath.Join(ctx.ImageConfigDir, k8sDir, k8sConfigDir)
	require.NoError(t, os.MkdirAll(configDir, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "server.yaml"), b, os.ModePerm))

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_EXEC=$NODETYPE")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/bin")
	assert.Contains(t, contents, "chmod +x $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.NotContains(t, contents, "sh set-node-ip.sh")

	// Server config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://192.168.122.100:6443", configContents["server"])
	assert.Equal(t, []any{"k8s-host.com", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Nil(t, configContents["cluster-init"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.Equal(t, []any{"k8s-host.com", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Equal(t, true, configContents["cluster-init"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "agent.yaml")

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

func TestConfigureKubernetes_Successful_MultiNode_K3s_IPv6(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
			APIVIP6: "fd12:3456:789a::21",
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

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadK3sArtefacts: func(arch image.Arch, version, installPath, imagesPath string) error {
				binary := filepath.Join(installPath, "cool-k3s-binary")
				return os.WriteFile(binary, nil, os.ModePerm)
			},
		},
	}

	serverConfig := map[string]any{
		"token": "123",
		"tls-san": []string{
			"k8s-host.com",
		},
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configDir := filepath.Join(ctx.ImageConfigDir, k8sDir, k8sConfigDir)
	require.NoError(t, os.MkdirAll(configDir, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "server.yaml"), b, os.ModePerm))

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "echo \"fd12:3456:789a::21 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_EXEC=$NODETYPE")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/bin")
	assert.Contains(t, contents, "chmod +x $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.Contains(t, contents, "sh set-node-ip.sh")

	// Server config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://[fd12:3456:789a::21]:6443", configContents["server"])
	assert.Equal(t, []any{"k8s-host.com", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Nil(t, configContents["cluster-init"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.Equal(t, []any{"k8s-host.com", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Equal(t, true, configContents["cluster-init"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "agent.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://[fd12:3456:789a::21]:6443", configContents["server"])
	assert.Nil(t, configContents["tls-san"])
	assert.Nil(t, configContents["disable"])
	assert.Nil(t, configContents["cluster-init"])
}

func TestConfigureKubernetes_Successful_MultiNode_K3s_Dualstack_PrioIPv4(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
			APIVIP4: "192.168.122.100",
			APIVIP6: "fd12:3456:789a::21",
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

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadK3sArtefacts: func(arch image.Arch, version, installPath, imagesPath string) error {
				binary := filepath.Join(installPath, "cool-k3s-binary")
				return os.WriteFile(binary, nil, os.ModePerm)
			},
		},
	}

	serverConfig := map[string]any{
		"token": "123",
		"tls-san": []string{
			"k8s-host.com",
		},
		"cluster-cidr": "10.42.0.0/16,fd12:3456:789b::/48",
		"service-cidr": "10.43.0.0/16,fd12:3456:789c::/112",
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configDir := filepath.Join(ctx.ImageConfigDir, k8sDir, k8sConfigDir)
	require.NoError(t, os.MkdirAll(configDir, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "server.yaml"), b, os.ModePerm))

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "echo \"fd12:3456:789a::21 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_EXEC=$NODETYPE")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/bin")
	assert.Contains(t, contents, "chmod +x $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.Contains(t, contents, "sh set-node-ip.sh")

	// Server config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://192.168.122.100:6443", configContents["server"])
	assert.ElementsMatch(t, []any{"k8s-host.com", "192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Nil(t, configContents["cluster-init"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.ElementsMatch(t, []any{"k8s-host.com", "192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Equal(t, true, configContents["cluster-init"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "agent.yaml")

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

func TestConfigureKubernetes_Successful_MultiNode_K3s_Dualstack_PrioIPv6(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
			APIVIP4: "192.168.122.100",
			APIVIP6: "fd12:3456:789a::21",
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

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadK3sArtefacts: func(arch image.Arch, version, installPath, imagesPath string) error {
				binary := filepath.Join(installPath, "cool-k3s-binary")
				return os.WriteFile(binary, nil, os.ModePerm)
			},
		},
	}

	serverConfig := map[string]any{
		"token": "123",
		"tls-san": []string{
			"k8s-host.com",
		},
		"cluster-cidr": "fd12:3456:789b::/48,10.42.0.0/16",
		"service-cidr": "fd12:3456:789c::/112,10.43.0.0/16",
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configDir := filepath.Join(ctx.ImageConfigDir, k8sDir, k8sConfigDir)
	require.NoError(t, os.MkdirAll(configDir, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "server.yaml"), b, os.ModePerm))

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/k3s/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/k3s/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "echo \"fd12:3456:789a::21 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_K3S_EXEC=$NODETYPE")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_DOWNLOAD=true")
	assert.Contains(t, contents, "export INSTALL_K3S_SKIP_START=true")
	assert.Contains(t, contents, "export INSTALL_K3S_BIN_DIR=/opt/bin")
	assert.Contains(t, contents, "chmod +x $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/install/cool-k3s-binary $INSTALL_K3S_BIN_DIR/k3s")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.Contains(t, contents, "sh set-node-ip.sh")

	// Server config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://[fd12:3456:789a::21]:6443", configContents["server"])
	assert.ElementsMatch(t, []any{"k8s-host.com", "192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Nil(t, configContents["cluster-init"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.ElementsMatch(t, []any{"k8s-host.com", "192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
	assert.Equal(t, []any{"servicelb"}, configContents["disable"])
	assert.Equal(t, true, configContents["cluster-init"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "agent.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://[fd12:3456:789a::21]:6443", configContents["server"])
	assert.Nil(t, configContents["tls-san"])
	assert.Nil(t, configContents["disable"])
	assert.Nil(t, configContents["cluster-init"])
}

func TestConfigureKubernetes_Successful_SingleNode_RKE2_IPv4(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIVIP4: "192.168.122.100",
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
		},
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, ingressController string, installPath, imagesPath string) error {
				return nil
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/server.yaml /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=$ARTEFACTS_DIR/kubernetes/install")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.Contains(t, contents, "systemctl enable rke2-server.service")
	assert.NotContains(t, contents, "sh set-node-ip.sh")

	// Config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

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

func TestConfigureKubernetes_Successful_MultiNode_RKE2_Dualstack_PrioIPv6_WithSingleNodeIP(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
			APIVIP4: "192.168.122.100",
			APIVIP6: "fd12:3456:789a::21",
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

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, ingressController string, installPath, imagesPath string) error {
				return nil
			},
		},
	}

	serverConfig := map[string]any{
		"token": "123",
		"cni":   "canal",
		"tls-san": []string{
			"k8s-host.com",
		},
		"cluster-cidr": "fd12:3456:789b::/48,10.42.0.0/16",
		"service-cidr": "fd12:3456:789c::/112,10.43.0.0/16",
		"node-ip":      "fd12:3456:789a::21",
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configDir := filepath.Join(ctx.ImageConfigDir, k8sDir, k8sConfigDir)
	require.NoError(t, os.MkdirAll(configDir, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "server.yaml"), b, os.ModePerm))

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "echo \"fd12:3456:789a::21 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=$ARTEFACTS_DIR/kubernetes/install")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.Contains(t, contents, "systemctl enable rke2-$NODETYPE.service")
	assert.NotContains(t, contents, "sh set-node-ip.sh")

	// Server config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://[fd12:3456:789a::21]:9345", configContents["server"])
	assert.ElementsMatch(t, []any{"k8s-host.com", "192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.ElementsMatch(t, []any{"k8s-host.com", "192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "agent.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://[fd12:3456:789a::21]:9345", configContents["server"])
	assert.Nil(t, configContents["tls-san"])
}

func TestConfigureKubernetes_Successful_MultiNode_RKE2_Dualstack_PrioIPv6_WithDualstackNodeIP(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
			APIVIP4: "192.168.122.100",
			APIVIP6: "fd12:3456:789a::21",
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

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return kubernetesScriptInstaller, nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, ingressController string, installPath, imagesPath string) error {
				return nil
			},
		},
	}

	serverConfig := map[string]any{
		"token": "123",
		"cni":   "canal",
		"tls-san": []string{
			"k8s-host.com",
		},
		"cluster-cidr": "fd12:3456:789b::/48,10.42.0.0/16",
		"service-cidr": "fd12:3456:789c::/112,10.43.0.0/16",
		"node-ip":      "fd12:3456:789a::21,192.168.122.101",
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configDir := filepath.Join(ctx.ImageConfigDir, k8sDir, k8sConfigDir)
	require.NoError(t, os.MkdirAll(configDir, os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "server.yaml"), b, os.ModePerm))

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp $CONFIGFILE /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "if [ \"$HOSTNAME\" = node1.suse.com ]; then")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "echo \"fd12:3456:789a::21 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=$ARTEFACTS_DIR/kubernetes/install")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-kubernetes.sh")
	assert.Contains(t, contents, "systemctl enable rke2-$NODETYPE.service")
	assert.NotContains(t, contents, "sh set-node-ip.sh")

	// Server config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

	info, err = os.Stat(configPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	var configContents map[string]any
	require.NoError(t, yaml.Unmarshal(b, &configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://[fd12:3456:789a::21]:9345", configContents["server"])
	assert.ElementsMatch(t, []any{"k8s-host.com", "192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.ElementsMatch(t, []any{"k8s-host.com", "192.168.122.100", "fd12:3456:789a::21", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.ArtefactsDir, "kubernetes", "agent.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://[fd12:3456:789a::21]:9345", configContents["server"])
	assert.Nil(t, configContents["tls-san"])
}

func TestConfigureManifests_NoSetup(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	var c Combustion

	manifestsPath, err := c.configureManifests(ctx)
	require.NoError(t, err)

	assert.Equal(t, "", manifestsPath)
}

func TestConfigureManifests_InvalidManifestDir(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	c := Combustion{
		Registry: &mockEmbeddedRegistry{
			manifestsPathFunc: func() string {
				return "non-existing"
			},
		},
	}

	_, err := c.configureManifests(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "copying manifests to combustion dir: reading source dir: open non-existing: no such file or directory")
}

func TestConfigureManifests_HelmChartsError(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	c := Combustion{
		Registry: &mockEmbeddedRegistry{
			manifestsPathFunc: func() string {
				// Use local test files
				return filepath.Join("..", "registry", "testdata")
			},
			helmChartsFunc: func() ([]*registry.HelmCRD, error) {
				return nil, fmt.Errorf("some error")
			},
		},
	}

	_, err := c.configureManifests(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "getting helm charts: some error")
}

func TestConfigureManifests(t *testing.T) {
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

	c := Combustion{
		Registry: &mockEmbeddedRegistry{
			manifestsPathFunc: func() string {
				// Use local test files
				return filepath.Join("..", "registry", "testdata")
			},
			helmChartsFunc: func() ([]*registry.HelmCRD, error) {
				return []*registry.HelmCRD{
					registry.NewHelmCRD(helmChart, "some-content", `
values: content`, "oci://registry-1.docker.io/bitnamicharts"),
				}, nil
			},
		},
	}

	manifestsPath, err := c.configureManifests(ctx)
	require.NoError(t, err)

	assert.Equal(t, "$ARTEFACTS_DIR/kubernetes/manifests", manifestsPath)

	manifestPath := filepath.Join(ctx.ArtefactsDir, k8sDir, k8sManifestsDir, "sample-crd.yaml")

	b, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	contents := string(b)
	assert.Contains(t, contents, "apiVersion: apps/v1")
	assert.Contains(t, contents, "kind: Deployment")
	assert.Contains(t, contents, "name: my-nginx")
	assert.Contains(t, contents, "image: nginx:1.14.2")

	chartPath := filepath.Join(ctx.ArtefactsDir, k8sDir, k8sManifestsDir, "apache.yaml")
	chartContent := `apiVersion: helm.cattle.io/v1
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
    backOffLimit: 20
`
	b, err = os.ReadFile(chartPath)
	require.NoError(t, err)

	assert.Equal(t, chartContent, string(b))
}

func TestConfigureKubernetes_Successful_RKE2Server_WithManifests(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIVIP4: "192.168.122.100",
			APIHost: "api.cluster01.hosted.on.edge.suse.com",
		},
	}

	c := Combustion{
		KubernetesScriptDownloader: mockKubernetesScriptDownloader{
			downloadScript: func(distribution, destPath string) (string, error) {
				return "install-k8s.sh", nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadRKE2Artefacts: func(arch image.Arch, version, cni string, multusEnabled bool, ingressController string, installPath, imagesPath string) error {
				return nil
			},
		},
		Registry: mockEmbeddedRegistry{
			manifestsPathFunc: func() string {
				// Use local test files
				return filepath.Join("..", "registry", "testdata")
			},
			helmChartsFunc: func() ([]*registry.HelmCRD, error) {
				return nil, nil
			},
		},
	}

	scripts, err := c.configureKubernetes(ctx)
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
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/server.yaml /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "echo \"192.168.122.100 api.cluster01.hosted.on.edge.suse.com\" >> /etc/hosts")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=$ARTEFACTS_DIR/kubernetes/install")
	assert.Contains(t, contents, "sh $ARTEFACTS_DIR/kubernetes/install-k8s.sh")
	assert.Contains(t, contents, "systemctl enable rke2-server.service")
	assert.Contains(t, contents, "mkdir -p /opt/eib-k8s/manifests")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/manifests/* /opt/eib-k8s/manifests/")
	assert.Contains(t, contents, "cp $ARTEFACTS_DIR/kubernetes/registries.yaml /etc/rancher/rke2/registries.yaml")
	assert.NotContains(t, contents, "sh set-node-ip.sh")

	// Config file assertions
	configPath := filepath.Join(ctx.ArtefactsDir, "kubernetes", "server.yaml")

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

	// Manifest assertions
	manifest := filepath.Join(ctx.ArtefactsDir, k8sDir, k8sManifestsDir, "sample-crd.yaml")
	info, err = os.Stat(manifest)
	require.NoError(t, err)
	assert.Equal(t, fileio.NonExecutablePerms, info.Mode())

	b, err = os.ReadFile(manifest)
	require.NoError(t, err)

	contents = string(b)
	assert.Contains(t, contents, "apiVersion: apps/v1")
	assert.Contains(t, contents, "kind: Deployment")
	assert.Contains(t, contents, "name: my-nginx")
	assert.Contains(t, contents, "image: nginx:1.14.2")
}

func TestKubernetesVIPManifestValidIPV4(t *testing.T) {
	k8s := &image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIVIP4: "192.168.1.1",
		},
	}

	manifest, err := kubernetesVIPManifest(k8s)
	require.NoError(t, err)

	assert.Contains(t, manifest, "- 192.168.1.1/32")
	assert.Contains(t, manifest, "- name: rke2-api")
	assert.NotContains(t, manifest, "ipFamilies:\n      - IPv6")
	assert.NotContains(t, manifest, "ipFamilyPolicy: SingleStack")
	assert.NotContains(t, manifest, "ipFamilyPolicy: RequireDualStack")
}

func TestKubernetesVIPManifestValidIPV6(t *testing.T) {
	k8s := &image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIVIP6: "fd12:3456:789a::21",
		},
	}

	manifest, err := kubernetesVIPManifest(k8s)
	require.NoError(t, err)

	assert.Contains(t, manifest, "- fd12:3456:789a::21/128")
	assert.Contains(t, manifest, "- name: k8s-api")
	assert.Contains(t, manifest, "ipFamilies:\n    - IPv6")
	assert.Contains(t, manifest, "ipFamilyPolicy: SingleStack")
	assert.NotContains(t, manifest, "ipFamilyPolicy: RequireDualStack")
	assert.NotContains(t, manifest, "rke2")
}

func TestKubernetesVIPManifestDualstack(t *testing.T) {
	k8s := &image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIVIP4: "192.168.1.1",
			APIVIP6: "fd12:3456:789a::21",
		},
	}

	manifest, err := kubernetesVIPManifest(k8s)
	require.NoError(t, err)

	assert.Contains(t, manifest, "- 192.168.1.1/32")
	assert.Contains(t, manifest, "- fd12:3456:789a::21/128")
	assert.Contains(t, manifest, "- name: k8s-api")
	assert.NotContains(t, manifest, "ipFamilies:\n      - IPv6")
	assert.NotContains(t, manifest, "ipFamilyPolicy: SingleStack")
	assert.Contains(t, manifest, "ipFamilyPolicy: RequireDualStack")
	assert.Contains(t, manifest, "ipFamilies:\n    - IPv4\n    - IPv6\n")
}

func TestCreateNodeIPScript_Dualstack_K3s_PrioIPv6(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+k3s1",
		Network: image.Network{
			APIVIP4: "192.168.1.1",
			APIVIP6: "fd12:3456:789a::21",
		},
	}

	serverConfig := map[string]any{
		"cluster-cidr": "fd12:3456:789b::/48,10.42.0.0/16",
		"service-cidr": "fd12:3456:789c::/112,10.43.0.0/16",
	}

	nodeIPScript, err := createNodeIPScript(ctx, serverConfig)
	require.NoError(t, err)
	assert.Equal(t, "set-node-ip.sh", nodeIPScript)

	nodeIPScriptPath := filepath.Join(ctx.CombustionDir, setNodeIPScript)
	b, err := os.ReadFile(nodeIPScriptPath)
	require.NoError(t, err)

	contents := string(b)

	assert.Contains(t, contents, "IPv4=true")
	assert.Contains(t, contents, "prioritizeIPv6=true")
	assert.Contains(t, contents, "CONFIG_FILE=\"/etc/rancher/k3s/config.yaml\"")
}

func TestCreateNodeIPScript_Dualstack_Rke2_PrioIPv4(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIVIP4: "192.168.1.1",
			APIVIP6: "fd12:3456:789a::21",
		},
	}

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16,fd12:3456:789b::/48",
		"service-cidr": "10.43.0.0/16,fd12:3456:789c::/112",
	}

	nodeIPScript, err := createNodeIPScript(ctx, serverConfig)
	require.NoError(t, err)
	assert.Equal(t, "set-node-ip.sh", nodeIPScript)

	nodeIPScriptPath := filepath.Join(ctx.CombustionDir, setNodeIPScript)
	b, err := os.ReadFile(nodeIPScriptPath)
	require.NoError(t, err)

	contents := string(b)

	assert.Contains(t, contents, "IPv4=true")
	assert.Contains(t, contents, "prioritizeIPv6=false")
	assert.Contains(t, contents, "CONFIG_FILE=\"/etc/rancher/rke2/config.yaml\"")
}

func TestCreateNodeIPScript_IPv4Only(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIVIP4: "192.168.1.1",
		},
	}

	serverConfig := map[string]any{}

	nodeIPScript, err := createNodeIPScript(ctx, serverConfig)
	require.NoError(t, err)
	assert.Empty(t, nodeIPScript)
}

func TestCreateNodeIPScript_IPv6Only(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIVIP6: "fd12:3456:789a::21",
		},
	}

	serverConfig := map[string]any{
		"cluster-cidr": "fd12:3456:789b::/48",
		"service-cidr": "fd12:3456:789c::/112",
	}

	nodeIPScript, err := createNodeIPScript(ctx, serverConfig)
	require.NoError(t, err)
	assert.Equal(t, "set-node-ip.sh", nodeIPScript)

	nodeIPScriptPath := filepath.Join(ctx.CombustionDir, setNodeIPScript)
	b, err := os.ReadFile(nodeIPScriptPath)
	require.NoError(t, err)

	contents := string(b)
	assert.Contains(t, contents, "CONFIG_FILE=\"/etc/rancher/rke2/config.yaml\"")
	assert.Contains(t, contents, "IPv4=false")
	assert.Contains(t, contents, "prioritizeIPv6=true")
}

func TestCreateNodeIPScript_NodeIPSpecified(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version: "v1.30.3+rke2r1",
		Network: image.Network{
			APIVIP4: "192.168.1.1",
		},
	}

	serverConfig := map[string]any{
		"node-ip": "192.168.100.100",
	}

	nodeIPScript, err := createNodeIPScript(ctx, serverConfig)
	require.NoError(t, err)
	assert.Empty(t, nodeIPScript)
}
