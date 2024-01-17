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

func TestConfigureKubernetes_ConfigFileMissingErrorRKE2(t *testing.T) {
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
			return "", "", nil
		},
	}

	require.NoError(t, os.Mkdir(filepath.Join(ctx.ImageConfigDir, k8sConfigDir), os.ModePerm))

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: parsing RKE2 config: "+
		"kubernetes component directory exists but does not contain config.yaml")
	assert.Nil(t, scripts)
}

func TestConfigureKubernetes_SuccessfulRKE2Server(t *testing.T) {
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
			return "server-installer", "server-images", nil
		},
	}

	configDir := filepath.Join(ctx.ImageConfigDir, k8sConfigDir)
	require.NoError(t, os.Mkdir(configDir, os.ModePerm))

	configFile := filepath.Join(configDir, k8sConfigFile)
	data := []byte("") // default CNI will be used since the file will not contain one
	require.NoError(t, os.WriteFile(configFile, data, os.ModePerm))

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
}

func TestConfigureKubernetes_SuccessfulRKE2Agent(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.Kubernetes = image.Kubernetes{
		Version:  "v1.29.0+rke2r1",
		NodeType: "agent",
	}
	ctx.KubernetesScriptInstaller = mockKubernetesScriptInstaller{
		installScript: func(distribution, sourcePath, destPath string) error {
			return nil
		},
	}
	ctx.KubernetesArtefactDownloader = mockKubernetesArtefactDownloader{
		downloadArtefacts: func(arch image.Arch, version, cni string, multusEnabled bool, destPath string) (string, string, error) {
			return "agent-installer", "agent-images", nil
		},
	}

	configDir := filepath.Join(ctx.ImageConfigDir, k8sConfigDir)
	require.NoError(t, os.Mkdir(configDir, os.ModePerm))

	data := map[string]any{
		"cni":   []string{"calico"},
		"debug": true,
	}
	b, err := yaml.Marshal(data)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, k8sConfigFile)
	require.NoError(t, os.WriteFile(configFile, b, os.ModePerm))

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
	assert.Contains(t, contents, "cp agent-images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "export INSTALL_RKE2_TYPE=agent")
	assert.Contains(t, contents, "cp rke2_config.yaml /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=agent-installer")
	assert.Contains(t, contents, "systemctl enable rke2-agent.service")

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
	assert.Equal(t, []any{"calico"}, configContents["cni"])
	assert.Equal(t, true, configContents["debug"])
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
