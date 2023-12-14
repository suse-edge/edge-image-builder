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
	assert.Equal(t, "iso", definition.Image.ImageType)

	// - Image
	assert.Equal(t, "slemicro5.5.iso", definition.Image.BaseImage)
	assert.Equal(t, "eibimage.iso", definition.Image.OutputImageName)

	// - Operating System -> Kernel Arguments
	expectedKernelArgs := []string{
		"alpha=foo",
		"beta=bar",
	}
	assert.Equal(t, expectedKernelArgs, definition.OperatingSystem.KernelArgs)

	// Operating System -> Users
	userConfigs := definition.OperatingSystem.Users
	require.Len(t, userConfigs, 3)
	assert.Equal(t, "alpha", userConfigs[0].Username)
	assert.Equal(t, "$6$bZfTI3Wj05fdxQcB$W1HJQTKw/MaGTCwK75ic9putEquJvYO7vMnDBVAfuAMFW58/79abky4mx9.8znK0UZwSKng9dVosnYQR1toH71", userConfigs[0].Password)
	assert.Contains(t, userConfigs[0].SSHKey, "ssh-rsa AAAAB3")
	assert.Equal(t, "beta", userConfigs[1].Username)
	assert.Equal(t, "$6$GHjiVHm2AT.Qxznz$1CwDuEBM1546E/sVE1Gn1y4JoGzW58wrckyx3jj2QnphFmceS6b/qFtkjw1cp7LSJNW1OcLe/EeIxDDHqZU6o1", userConfigs[1].Password)
	assert.Equal(t, "", userConfigs[1].SSHKey)
	assert.Equal(t, "gamma", userConfigs[2].Username)
	assert.Equal(t, "", userConfigs[2].Password)
	assert.Contains(t, userConfigs[2].SSHKey, "ssh-rsa BBBBB3")

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
	assert.Equal(t, false, suma.GetSSL)

	// Operating System -> Packages
	pkgConfig := definition.OperatingSystem.Packages
	require.Len(t, pkgConfig.PKGList, 6)
	require.Len(t, pkgConfig.AddRepos, 3)
	expectedPKGList := []string{
		"wget2",
		"dpdk22",
		"dpdk22-tools",
		"libdpdk-23",
		"libatomic1",
		"libbpf0",
	}
	assert.Equal(t, expectedPKGList, pkgConfig.PKGList)
	expectedAddRepos := []string{
		"http://updates.ext.suse.de/SUSE/Products/SLE-Module-Server-Applications/15-SP5/x86_64/product/",
		"http://updates.ext.suse.de/SUSE/Updates/SLE-Module-Basesystem/15-SP5/x86_64/update/",
		"http://updates.ext.suse.de/SUSE/Products/SLE-Module-Basesystem/15-SP5/x86_64/product/",
	}
	assert.Equal(t, expectedAddRepos, pkgConfig.AddRepos)
	assert.Equal(t, "INTERNAL-USE-ONLY-foo-bar", pkgConfig.RegCode)
}

func TestParseBadConfig(t *testing.T) {
	// Setup
	badData := []byte("Not actually YAML")

	// Test
	_, err := ParseDefinition(badData)

	// Verify
	require.Error(t, err)
	assert.ErrorContains(t, err, "could not parse the image definition")
}
