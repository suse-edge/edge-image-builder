package build

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestGenerateGRUBGuestfishCommands(t *testing.T) {
	// Setup
	builder := Builder{
		context: &image.Context{
			ImageDefinition: &image.Definition{
				OperatingSystem: image.OperatingSystem{
					KernelArgs: []string{"alpha", "beta"},
				},
			},
		},
	}

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
	builder := Builder{
		context: &image.Context{
			ImageDefinition: &image.Definition{
				OperatingSystem: image.OperatingSystem{},
			},
		},
	}

	// Test
	commandString, err := builder.generateGRUBGuestfishCommands()

	// Verify
	require.NoError(t, err)
	assert.Equal(t, "", commandString)
}
