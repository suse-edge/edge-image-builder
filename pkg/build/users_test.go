package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestConfigureUsers(t *testing.T) {
	// Setup
	imageConfig := &config.ImageConfig{
		OperatingSystem: config.OperatingSystem{
			Users: []config.OperatingSystemUser{
				{
					Username: "alpha",
					Password: "alpha123",
					SSHKey:   "alphakey",
				},
				{
					Username: "beta",
					Password: "beta123",
				},
				{
					Username: "gamma",
					SSHKey:   "gammakey",
				},
				{
					Username: "root",
					Password: "root123",
					SSHKey:   "rootkey",
				},
			},
		},
	}

	context, err := NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := &Builder{
		imageConfig: imageConfig,
		context:     context,
	}

	// Test
	err = builder.configureUsers()

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(context.CombustionDir, usersScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(userScriptMode), stats.Mode())

	foundContents := string(foundBytes)

	// - All fields specified
	assert.Contains(t, foundContents, "useradd -m alpha")
	assert.Contains(t, foundContents, "echo 'alpha:alpha123' | chpasswd -e\n")
	assert.Contains(t, foundContents, "mkdir -pm700 /home/alpha/.ssh/")
	assert.Contains(t, foundContents, "echo 'alphakey' >> /home/alpha/.ssh/authorized_keys")
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
	assert.Contains(t, foundContents, "echo 'rootkey' >> /root/.ssh/authorized_keys")
	assert.NotContains(t, foundContents, "chown -R root")
}

func TestConfigureUsers_NoUsers(t *testing.T) {
	// Setup
	context, err := NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := &Builder{
		imageConfig: &config.ImageConfig{},
		context:     context,
	}

	// Test
	err = builder.configureUsers()

	// Verify
	require.NoError(t, err)

	expectedFilename := filepath.Join(context.CombustionDir, usersScriptName)
	_, err = os.ReadFile(expectedFilename)
	require.ErrorIs(t, err, os.ErrNotExist)
}
