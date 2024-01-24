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
			"192.168.122.100.sslip.io",
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
	assert.Equal(t, []any{"192.168.122.100.sslip.io", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Initialising server config file assertions
	configPath = filepath.Join(ctx.CombustionDir, "init_server.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, nil, configContents["server"])
	assert.Equal(t, []any{"192.168.122.100.sslip.io", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])

	// Agent config file assertions
	configPath = filepath.Join(ctx.CombustionDir, "agent.yaml")

	b, err = os.ReadFile(configPath)
	require.NoError(t, err)

	configContents = map[string]any{} // clear the map
	require.NoError(t, yaml.Unmarshal(b, configContents))

	assert.Equal(t, "canal", configContents["cni"])
	assert.Equal(t, "123", configContents["token"])
	assert.Equal(t, "https://192.168.122.100:9345", configContents["server"])
	assert.Equal(t, []any{"192.168.122.100.sslip.io", "192.168.122.100", "api.cluster01.hosted.on.edge.suse.com"}, configContents["tls-san"])
}

func TestExtractCNI(t *testing.T) {
	tests := map[string]struct {
		input                 map[string]any
		expectedCNI           string
		expectedMultusEnabled bool
		expectedErr           string
	}{
		"CNI not configured": {
			input:       map[string]any{},
			expectedErr: "invalid cni: <nil>",
		},
		"Empty CNI string": {
			input: map[string]any{
				"cni": "",
			},
			expectedErr: "cni not configured",
		},
		"Empty CNI list": {
			input: map[string]any{
				"cni": []string{},
			},
			expectedErr: "invalid cni value: []",
		},
		"Multiple CNI list": {
			input: map[string]any{
				"cni": []string{"canal", "calico", "cilium"},
			},
			expectedErr: "invalid cni value: [canal calico cilium]",
		},
		"Valid CNI string": {
			input: map[string]any{
				"cni": "calico",
			},
			expectedCNI: "calico",
		},
		"Valid CNI list": {
			input: map[string]any{
				"cni": []string{"calico"},
			},
			expectedCNI: "calico",
		},
		"Valid CNI string with multus": {
			input: map[string]any{
				"cni": "multus, calico",
			},
			expectedCNI:           "calico",
			expectedMultusEnabled: true,
		},
		"Valid CNI list with multus": {
			input: map[string]any{
				"cni": []string{"multus", "calico"},
			},
			expectedCNI:           "calico",
			expectedMultusEnabled: true,
		},
		"Invalid standalone multus": {
			input: map[string]any{
				"cni": "multus",
			},
			expectedErr: "multus must be used alongside another primary cni selection",
		},
		"Invalid standalone multus list": {
			input: map[string]any{
				"cni": []string{"multus"},
			},
			expectedErr: "multus must be used alongside another primary cni selection",
		},
		"Valid CNI with invalid multus placement": {
			input: map[string]any{
				"cni": "cilium, multus",
			},
			expectedErr: "multiple cni values are only allowed if multus is the first one",
		},
		"Valid CNI list with invalid multus placement": {
			input: map[string]any{
				"cni": []string{"cilium", "multus"},
			},
			expectedErr: "multiple cni values are only allowed if multus is the first one",
		},
		"Invalid CNI list": {
			input: map[string]any{
				"cni": []any{"cilium", 6},
			},
			expectedErr: "invalid cni value: 6",
		},
		"Invalid CNI format": {
			input: map[string]any{
				"cni": 6,
			},
			expectedErr: "invalid cni: 6",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := yaml.Marshal(test.input)
			require.NoError(t, err)

			var config map[string]any
			require.NoError(t, yaml.Unmarshal(b, &config))

			cni, multusEnabled, err := extractCNI(config)

			if test.expectedErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedErr)
				assert.False(t, multusEnabled)
				assert.Empty(t, cni)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedCNI, cni)
				assert.Equal(t, test.expectedMultusEnabled, multusEnabled)
			}
		})
	}
}

func TestFindKubernetesInitialiserNode(t *testing.T) {
	tests := []struct {
		name         string
		nodes        []image.Node
		expectedNode string
	}{
		{
			name:         "Empty list of nodes",
			expectedNode: "",
		},
		{
			name: "Agent list",
			nodes: []image.Node{
				{
					Hostname: "host1",
					Type:     "agent",
				},
				{
					Hostname: "host2",
					Type:     "agent",
				},
			},

			expectedNode: "",
		},
		{
			name: "Server node labeled as initialiser",
			nodes: []image.Node{
				{
					Hostname: "host1",
					Type:     "agent",
				},
				{
					Hostname: "host2",
					Type:     "server",
				},
				{
					Hostname:    "host3",
					Type:        "server",
					Initialiser: true,
				},
			},
			expectedNode: "host3",
		},
		{
			name: "Initialiser as first server node in list",
			nodes: []image.Node{
				{
					Hostname: "host1",
					Type:     "agent",
				},
				{
					Hostname: "host2",
					Type:     "server",
				},
				{
					Hostname: "host3",
					Type:     "server",
				},
			},
			expectedNode: "host2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubernetes := &image.Kubernetes{Nodes: test.nodes}
			assert.Equal(t, test.expectedNode, findKubernetesInitialiserNode(kubernetes))
		})
	}
}

func TestAppendClusterTLSSAN(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]any
		apiHost        string
		expectedTLSSAN any
	}{
		{
			name:           "Empty TLS SAN",
			config:         map[string]any{},
			apiHost:        "",
			expectedTLSSAN: nil,
		},
		{
			name:           "Missing TLS SAN",
			config:         map[string]any{},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []string{"api.cluster01.hosted.on.edge.suse.com"},
		},
		{
			name: "Invalid TLS SAN",
			config: map[string]any{
				"tls-san": 5,
			},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []string{"api.cluster01.hosted.on.edge.suse.com"},
		},
		{
			name: "Existing TLS SAN string",
			config: map[string]any{
				"tls-san": "random",
			},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []string{"random", "api.cluster01.hosted.on.edge.suse.com"},
		},
		{
			name: "Existing TLS SAN string list",
			config: map[string]any{
				"tls-san": []string{"random"},
			},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []string{"random", "api.cluster01.hosted.on.edge.suse.com"},
		},
		{
			name: "Existing TLS SAN list",
			config: map[string]any{
				"tls-san": []any{"random"},
			},
			apiHost:        "api.cluster01.hosted.on.edge.suse.com",
			expectedTLSSAN: []any{"random", "api.cluster01.hosted.on.edge.suse.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			appendClusterTLSSAN(test.config, test.apiHost)
			assert.Equal(t, test.expectedTLSSAN, test.config["tls-san"])
		})
	}
}

func TestSetClusterAPIAddress(t *testing.T) {
	config := map[string]any{}

	setClusterAPIAddress(config, "")
	assert.NotContains(t, config, "server")

	setClusterAPIAddress(config, "192.168.122.50")
	assert.Equal(t, "https://192.168.122.50:9345", config["server"])
}
