package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestGenerateGRUBGuestfishCommands(t *testing.T) {
	// Setup
	imageConfig := config.ImageConfig{
		OperatingSystem: config.OperatingSystem{
			KernelArgs: []string{"alpha", "beta"},
		},
	}
	builder := New(&imageConfig, nil)

	// Test
	commandString, err := builder.generateGRUBGuestfishCommands()

	// Verify
	require.NoError(t, err)
	require.NotNil(t, commandString)

	expectedFirstBoot := "sed -i '/ignition.platform/ s/$/ alpha beta /' /tmp/grub.cfg"
	assert.Contains(t, commandString, expectedFirstBoot)

	expectedDefault := "sed -i '/^GRUB_CMDLINE_LINUX_DEFAULT=\"/ s/\"$/ alpha beta \"/' /tmp/grub"
	assert.Contains(t, commandString, expectedDefault)
}

func TestGenerateGRUBGuestfishCommandsNoArgs(t *testing.T) {
	// Setup
	imageConfig := config.ImageConfig{
		OperatingSystem: config.OperatingSystem{},
	}
	builder := New(&imageConfig, nil)

	// Test
	commandString, err := builder.generateGRUBGuestfishCommands()

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "", commandString)
}
