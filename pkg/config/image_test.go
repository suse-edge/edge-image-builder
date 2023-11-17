package config

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
	imageConfig, err := Parse(configData)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, "1.0", imageConfig.APIVersion)
	assert.Equal(t, "iso", imageConfig.Image.ImageType)
	assert.Equal(t, "slemicro5.5.iso", imageConfig.Image.BaseImage)
	assert.Equal(t, "eibimage.iso", imageConfig.Image.OutputImageName)

	expectedKernelArgs := []string{
		"alpha=foo",
		"beta=bar",
	}
	assert.Equal(t, expectedKernelArgs, imageConfig.OperatingSystem.KernelArgs)

	userConfigs := imageConfig.OperatingSystem.Users
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
}

func TestParseBadConfig(t *testing.T) {
	// Setup
	badData := []byte("Not actually YAML")

	// Test
	_, err := Parse(badData)

	// Verify
	require.Error(t, err)
	assert.ErrorContains(t, err, "could not parse the image configuration")
}
