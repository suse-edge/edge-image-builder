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
	downloadArtefacts func(kubernetes image.Kubernetes, arch image.Arch, destPath string) (string, string, error)
}

func (m mockKubernetesArtefactDownloader) DownloadArtefacts(kubernetes image.Kubernetes, arch image.Arch, destPath string) (installPath, imagesPath string, err error) {
	if m.downloadArtefacts != nil {
		return m.downloadArtefacts(kubernetes, arch, destPath)
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
	ctx := &image.Context{
		ImageDefinition: &image.Definition{
			Kubernetes: image.Kubernetes{
				Version: "v1.29.0+rke2r1",
			},
		},
		KubernetesScriptInstaller: mockKubernetesScriptInstaller{
			installScript: func(distribution, sourcePath, destPath string) error {
				return nil
			},
		},
		KubernetesArtefactDownloader: mockKubernetesArtefactDownloader{
			downloadArtefacts: func(kubernetes image.Kubernetes, arch image.Arch, destPath string) (string, string, error) {
				return "", "", fmt.Errorf("some error")
			},
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
		downloadArtefacts: func(kubernetes image.Kubernetes, arch image.Arch, destPath string) (string, string, error) {
			return "", "", nil
		},
	}

	require.NoError(t, os.Mkdir(filepath.Join(ctx.ImageConfigDir, k8sConfigDir), os.ModePerm))

	scripts, err := configureKubernetes(ctx)
	require.Error(t, err)
	assert.EqualError(t, err, "configuring kubernetes components: copying RKE2 config: "+
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
		downloadArtefacts: func(kubernetes image.Kubernetes, arch image.Arch, destPath string) (string, string, error) {
			return "server-installer", "server-images", nil
		},
	}

	configDir := filepath.Join(ctx.ImageConfigDir, k8sConfigDir)
	require.NoError(t, os.Mkdir(configDir, os.ModePerm))
	configFile := filepath.Join(configDir, k8sConfigFile)
	require.NoError(t, os.WriteFile(configFile, []byte("some-config-data"), os.ModePerm))

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

	contents = string(b)
	assert.Equal(t, "some-config-data", contents)
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
		downloadArtefacts: func(kubernetes image.Kubernetes, arch image.Arch, destPath string) (string, string, error) {
			return "agent-installer", "agent-images", nil
		},
	}

	scripts, err := configureKubernetes(ctx)
	require.NoError(t, err)
	require.Len(t, scripts, 1)

	scriptPath := filepath.Join(ctx.CombustionDir, scripts[0])

	info, err := os.Stat(scriptPath)
	require.NoError(t, err)

	assert.Equal(t, fileio.ExecutablePerms, info.Mode())

	b, err := os.ReadFile(scriptPath)
	require.NoError(t, err)

	contents := string(b)
	assert.Contains(t, contents, "cp agent-images/* /var/lib/rancher/rke2/agent/images/")
	assert.Contains(t, contents, "export INSTALL_RKE2_TYPE=agent")
	assert.NotContains(t, contents, "cp rke2_config.yaml /etc/rancher/rke2/config.yaml")
	assert.Contains(t, contents, "export INSTALL_RKE2_ARTIFACT_PATH=agent-installer")
	assert.Contains(t, contents, "systemctl enable rke2-agent.service")
}
