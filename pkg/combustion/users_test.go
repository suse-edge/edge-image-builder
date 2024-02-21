package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestConfigureUsers(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Users: []image.OperatingSystemUser{
				{
					Username:          "alpha",
					EncryptedPassword: "alpha123",
					SSHKeys:           []string{"alphakey1", "alphakey2"},
				},
				{
					Username:          "beta",
					EncryptedPassword: "beta123",
				},
				{
					Username: "gamma",
					SSHKeys:  []string{"gammakey"},
				},
				{
					Username:          "root",
					EncryptedPassword: "root123",
					SSHKeys:           []string{"rootkey1", "rootkey2"},
				},
			},
		},
	}

	// Test
	scripts, err := configureUsers(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, usersScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, usersScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - All fields specified
	assert.Contains(t, foundContents, "useradd -m alpha")
	assert.Contains(t, foundContents, "echo 'alpha:alpha123' | chpasswd -e\n")
	assert.Contains(t, foundContents, "mkdir -pm700 /home/alpha/.ssh/")
	assert.Contains(t, foundContents, "echo 'alphakey1' >> /home/alpha/.ssh/authorized_keys")
	assert.Contains(t, foundContents, "echo 'alphakey2' >> /home/alpha/.ssh/authorized_keys")
	assert.Contains(t, foundContents, "chown -R alpha /home/alpha/.ssh")

	// - Only a password set
	assert.Contains(t, foundContents, "useradd -m beta")
	assert.Contains(t, foundContents, "echo 'beta:beta123' | chpasswd -e\n")
	assert.NotContains(t, foundContents, "mkdir -pm700 /home/beta/.ssh/")
	assert.NotContains(t, foundContents, "/home/beta/.ssh/authorized_keys")
	assert.NotContains(t, foundContents, "chown -R beta /home/beta/.ssh")

	// - Only an SSH key specified
	assert.Contains(t, foundContents, "useradd -m gamma")
	assert.NotContains(t, foundContents, "echo 'gamma:")
	assert.Contains(t, foundContents, "mkdir -pm700 /home/gamma/.ssh/")
	assert.Contains(t, foundContents, "echo 'gammakey' >> /home/gamma/.ssh/authorized_keys")
	assert.Contains(t, foundContents, "chown -R gamma /home/gamma/.ssh")

	// - Special handling for root
	assert.NotContains(t, foundContents, "useradd -m root")
	assert.Contains(t, foundContents, "echo 'root:root123' | chpasswd -e\n")
	assert.Contains(t, foundContents, "mkdir -pm700 /root/.ssh/")
	assert.Contains(t, foundContents, "echo 'rootkey1' >> /root/.ssh/authorized_keys")
	assert.Contains(t, foundContents, "echo 'rootkey2' >> /root/.ssh/authorized_keys")
	assert.NotContains(t, foundContents, "chown -R root")
}

func TestConfigureUsers_NoUsers(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	scripts, err := configureUsers(ctx)

	// Verify
	require.NoError(t, err)

	assert.Len(t, scripts, 0)

	expectedFilename := filepath.Join(ctx.CombustionDir, usersScriptName)
	_, err = os.ReadFile(expectedFilename)
	require.ErrorIs(t, err, os.ErrNotExist)
}
