package image

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	// Setup
	filename := "./testdata/full-valid-example.yaml"
	configData, err := os.ReadFile(filename)
	require.NoError(t, err)

	// Test
	definition, err := ParseDefinition(configData)

	// Verify
	require.NoError(t, err)

	// - Definition
	assert.Equal(t, "1.0", definition.APIVersion)
	assert.EqualValues(t, "x86_64", definition.Image.Arch)
	assert.Equal(t, "iso", definition.Image.ImageType)

	// - Image
	assert.Equal(t, "slemicro5.5.iso", definition.Image.BaseImage)
	assert.Equal(t, "eibimage.iso", definition.Image.OutputImageName)

	// - Operating System -> Kernel Arguments
	expectedKernelArgs := []string{
		"alpha=foo",
		"beta=bar",
		"baz",
	}
	assert.Equal(t, expectedKernelArgs, definition.OperatingSystem.KernelArgs)

	// Operating System -> Groups
	groupConfigs := definition.OperatingSystem.Groups
	require.Len(t, groupConfigs, 2)
	assert.Equal(t, "group1", groupConfigs[0].Name)
	assert.Equal(t, "group2", groupConfigs[1].Name)

	// Operating System -> Users
	userConfigs := definition.OperatingSystem.Users
	require.Len(t, userConfigs, 3)
	assert.Equal(t, "alpha", userConfigs[0].Username)
	assert.Equal(t, 2000, userConfigs[0].UID)
	assert.Equal(t, "$6$bZfTI3Wj05fdxQcB$W1HJQTKw/MaGTCwK75ic9putEquJvYO7vMnDBVAfuAMFW58/79abky4mx9.8znK0UZwSKng9dVosnYQR1toH71", userConfigs[0].EncryptedPassword)
	assert.Len(t, userConfigs[0].SSHKeys, 2)
	assert.Contains(t, userConfigs[0].SSHKeys[0], "ssh-rsa AAAAB3")
	assert.Contains(t, userConfigs[0].SSHKeys[1], "ssh-rsa BBBBB3")
	assert.Equal(t, userConfigs[0].PrimaryGroup, "admin")
	assert.Len(t, userConfigs[0].SecondaryGroups, 1)
	assert.Equal(t, "wheel", userConfigs[0].SecondaryGroups[0])
	assert.True(t, userConfigs[0].CreateHomeDir)

	assert.Equal(t, "beta", userConfigs[1].Username)
	assert.Equal(t, 0, userConfigs[1].UID)
	assert.Equal(t, "$6$GHjiVHm2AT.Qxznz$1CwDuEBM1546E/sVE1Gn1y4JoGzW58wrckyx3jj2QnphFmceS6b/qFtkjw1cp7LSJNW1OcLe/EeIxDDHqZU6o1", userConfigs[1].EncryptedPassword)
	assert.Nil(t, userConfigs[1].SSHKeys)
	assert.Equal(t, userConfigs[1].PrimaryGroup, "")
	assert.Len(t, userConfigs[1].SecondaryGroups, 0)
	assert.False(t, userConfigs[1].CreateHomeDir)

	assert.Equal(t, "gamma", userConfigs[2].Username)
	assert.Equal(t, 0, userConfigs[2].UID)
	assert.Equal(t, "", userConfigs[2].EncryptedPassword)
	assert.Len(t, userConfigs[2].SSHKeys, 1)
	assert.Contains(t, userConfigs[2].SSHKeys[0], "ssh-rsa BBBBB3")
	assert.Equal(t, userConfigs[2].PrimaryGroup, "")
	assert.Len(t, userConfigs[2].SecondaryGroups, 0)
	assert.False(t, userConfigs[2].CreateHomeDir)

	// Operating System -> Systemd
	systemd := definition.OperatingSystem.Systemd
	require.Len(t, systemd.Enable, 2)
	assert.Equal(t, "enable0", systemd.Enable[0])
	assert.Equal(t, "enable1", systemd.Enable[1])
	require.Len(t, systemd.Disable, 1)
	assert.Equal(t, "disable0", systemd.Disable[0])

	// Operating System -> Suma
	suma := definition.OperatingSystem.Suma
	assert.Equal(t, "suma.edge.suse.com", suma.Host)
	assert.Equal(t, "slemicro55", suma.ActivationKey)

	// Operating System -> Packages
	pkgConfig := definition.OperatingSystem.Packages
	assert.True(t, pkgConfig.NoGPGCheck)
	require.Len(t, pkgConfig.PKGList, 6)
	require.Len(t, pkgConfig.AdditionalRepos, 2)
	expectedPKGList := []string{
		"wget2",
		"dpdk22",
		"dpdk22-tools",
		"libdpdk-23",
		"libatomic1",
		"libbpf0",
	}
	assert.Equal(t, expectedPKGList, pkgConfig.PKGList)
	expectedAddRepos := []AddRepo{
		{
			URL: "https://download.nvidia.com/suse/sle15sp5/",
		},
		{
			URL:      "https://developer.download.nvidia.com/compute/cuda/repos/sles15/x86_64/",
			Unsigned: true,
		},
	}
	assert.Equal(t, expectedAddRepos, pkgConfig.AdditionalRepos)
	assert.Equal(t, "INTERNAL-USE-ONLY-foo-bar", pkgConfig.RegCode)

	// Operating System -> IsoConfiguration
	installDevice := definition.OperatingSystem.IsoConfiguration.InstallDevice
	assert.Equal(t, "/dev/sda", installDevice)

	// Operating System -> Time
	time := definition.OperatingSystem.Time
	assert.Equal(t, "Europe/London", time.Timezone)
	expectedChronyPools := []string{
		"2.suse.pool.ntp.org",
	}
	assert.Equal(t, expectedChronyPools, time.NtpConfiguration.Pools)
	expectedChronyServers := []string{
		"10.0.0.1",
		"10.0.0.2",
	}
	assert.Equal(t, expectedChronyServers, time.NtpConfiguration.Servers)

	// Operating System -> Proxy -> HTTPProxy
	httpProxy := definition.OperatingSystem.Proxy.HTTPProxy
	assert.Equal(t, "http://10.0.0.1:3128", httpProxy)

	// Operating System -> Proxy -> HTTPSProxy
	httpsProxy := definition.OperatingSystem.Proxy.HTTPSProxy
	assert.Equal(t, "http://10.0.0.1:3128", httpsProxy)

	// Operating System -> Proxy -> NoProxy
	noProxy := definition.OperatingSystem.Proxy.NoProxy
	assert.Equal(t, []string{"localhost", "127.0.0.1", "edge.suse.com"}, noProxy)

	// Operating System -> Keymap
	keymap := definition.OperatingSystem.Keymap
	assert.Equal(t, "us", keymap)

	// EmbeddedArtifactRegistry
	embeddedArtifactRegistry := definition.EmbeddedArtifactRegistry
	assert.Equal(t, "hello-world:latest", embeddedArtifactRegistry.ContainerImages[0].Name)
	assert.Equal(t, "ghcr.io/fluxcd/flux-cli@sha256:02aa820c3a9c57d67208afcfc4bce9661658c17d15940aea369da259d2b976dd", embeddedArtifactRegistry.ContainerImages[1].Name)

	// Kubernetes
	kubernetes := definition.Kubernetes

	// Version
	assert.Equal(t, "v1.29.0+rke2r1", kubernetes.Version)

	// Network
	assert.Equal(t, "192.168.122.100", kubernetes.Network.APIVIP)
	assert.Equal(t, "api.cluster01.hosted.on.edge.suse.com", kubernetes.Network.APIHost)

	// Nodes
	require.Len(t, kubernetes.Nodes, 5)
	assert.Equal(t, "node1.suse.com", kubernetes.Nodes[0].Hostname)
	assert.Equal(t, "server", kubernetes.Nodes[0].Type)
	assert.Equal(t, false, kubernetes.Nodes[0].Initialiser)
	assert.Equal(t, "node2.suse.com", kubernetes.Nodes[1].Hostname)
	assert.Equal(t, "server", kubernetes.Nodes[1].Type)
	assert.Equal(t, true, kubernetes.Nodes[1].Initialiser)
	assert.Equal(t, "node3.suse.com", kubernetes.Nodes[2].Hostname)
	assert.Equal(t, "agent", kubernetes.Nodes[2].Type)
	assert.Equal(t, false, kubernetes.Nodes[2].Initialiser)
	assert.Equal(t, "node4.suse.com", kubernetes.Nodes[3].Hostname)
	assert.Equal(t, "server", kubernetes.Nodes[3].Type)
	assert.Equal(t, false, kubernetes.Nodes[4].Initialiser)
	assert.Equal(t, "node5.suse.com", kubernetes.Nodes[4].Hostname)
	assert.Equal(t, "agent", kubernetes.Nodes[4].Type)
	assert.Equal(t, false, kubernetes.Nodes[4].Initialiser)

	// Manifests
	assert.Equal(t, "https://k8s.io/examples/application/nginx-app.yaml", kubernetes.Manifests.URLs[0])

	// Helm Charts
	assert.Equal(t, "apache", kubernetes.Helm.Charts[0].Name)
	assert.Equal(t, "bitnami", kubernetes.Helm.Charts[0].RepositoryName)
	assert.Equal(t, "10.7.0", kubernetes.Helm.Charts[0].Version)
	assert.Equal(t, "web", kubernetes.Helm.Charts[0].TargetNamespace)
	assert.Equal(t, true, kubernetes.Helm.Charts[0].CreateNamespace)
	assert.Equal(t, "apache-system", kubernetes.Helm.Charts[0].InstallationNamespace)
	assert.Equal(t, "apache-values.yaml", kubernetes.Helm.Charts[0].ValuesFile)

	assert.Equal(t, "metallb", kubernetes.Helm.Charts[1].Name)
	assert.Equal(t, "suse-edge", kubernetes.Helm.Charts[1].RepositoryName)
	assert.Equal(t, "0.14.3", kubernetes.Helm.Charts[1].Version)

	// Helm Repositories
	assert.Equal(t, "suse-edge", kubernetes.Helm.Repositories[0].Name)
	assert.Equal(t, "https://suse-edge.github.io/charts", kubernetes.Helm.Repositories[0].URL)
	assert.Equal(t, "suse-edge.crt", kubernetes.Helm.Repositories[0].CAFile)

	assert.Equal(t, "bitnami", kubernetes.Helm.Repositories[1].Name)
	assert.Equal(t, "oci://registry-1.docker.io/bitnamicharts", kubernetes.Helm.Repositories[1].URL)
	assert.Equal(t, "user", kubernetes.Helm.Repositories[1].Authentication.Username)
	assert.Equal(t, "pass", kubernetes.Helm.Repositories[1].Authentication.Password)
	assert.Equal(t, false, kubernetes.Helm.Repositories[1].PlainHTTP)
	assert.Equal(t, true, kubernetes.Helm.Repositories[1].SkipTLSVerify)
}

func TestParseBadConfig_InvalidFormat(t *testing.T) {
	// Setup
	badData := []byte("Not actually YAML")

	// Test
	_, err := ParseDefinition(badData)

	// Verify
	require.Error(t, err)
	assert.ErrorContains(t, err, "could not parse the image definition")
	assert.ErrorContains(t, err, "line 1: cannot unmarshal !!str")
}

func TestParseBadConfig_UnknownFields(t *testing.T) {
	badConfig := `
apiVersion: 1.0
image:
  type: iso
operatingSystem:
  time:
    zone: Europe/London
`

	_, err := ParseDefinition([]byte(badConfig))

	require.Error(t, err)
	assert.ErrorContains(t, err, "could not parse the image definition")
	assert.ErrorContains(t, err, "line 4: field type not found in type image.Image")
	assert.ErrorContains(t, err, "line 7: field zone not found in type image.Time")
}

func TestArch_Short(t *testing.T) {
	assert.Equal(t, "amd64", ArchTypeX86.Short())
	assert.Equal(t, "arm64", ArchTypeARM.Short())
	assert.PanicsWithValue(t, "unknown arch: abc", func() {
		arch := Arch("abc")
		arch.Short()
	})
}

func TestDiskSize_ToMB(t *testing.T) {
	assert.EqualValues(t, 50, DiskSize("50M").ToMB())
	assert.EqualValues(t, 4096, DiskSize("4G").ToMB())
	assert.EqualValues(t, 1024*1024, DiskSize("1T").ToMB())

	assert.PanicsWithValue(t, "unknown disk size format", func() {
		DiskSize("10K").ToMB()
	})
}
