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

func TestConfigureTimesync(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Time: image.Time{
				Timezone: "Europe/London",
				NtpConfiguration: image.NtpConfiguration{
					Pools:     []string{"2.suse.pool.ntp.org"},
					Servers:   []string{"10.0.0.1", "10.0.0.2"},
					ForceWait: true,
				},
			},
		},
	}

	// Test
	scripts, err := configureTimesync(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, timesyncScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, timesyncScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - Ensure that we're creating the firstboot-timesync service
	assert.Contains(t, foundContents, "/etc/systemd/system/firstboot-timesync.service")

	// - Ensure that we've got the chrony-wait service starting at boot
	assert.Contains(t, foundContents, "systemctl enable chrony-wait")
}
