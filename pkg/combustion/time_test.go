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

func TestConfigureTime_NoConf(t *testing.T) {
	// Setup
	var ctx image.Context

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Time: image.Time{},
		},
	}

	// Test
	scripts, err := configureTime(&ctx)

	// Verify
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureTime_FullConfiguration(t *testing.T) {
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
	scripts, err := configureTime(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, timeScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, timeScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - Make sure that the symbolic link is created with correct timezone
	assert.Contains(t, foundContents, "ln -sf /usr/share/zoneinfo/Europe/London /etc/localtime", "symbolic link not created")

	// - Ensure that we have the correct chrony pool listed in chrony sources
	assert.Contains(t, foundContents, "pool 2.suse.pool.ntp.org iburst", "chrony pool not created")

	// - Ensure that we have the correct first chrony server listed in chrony sources
	assert.Contains(t, foundContents, "server 10.0.0.1 iburst", "first chronyServer not created")

	// - Ensure that we have the correct second chrony server listed in chrony sources
	assert.Contains(t, foundContents, "server 10.0.0.1 iburst", "second chronyServer not created")

	// - Ensure that we're creating the firstboot-timesync service
	assert.Contains(t, foundContents, "/etc/systemd/system/firstboot-timesync.service")

	// - Ensure that we've got the chrony-wait service starting at boot
	assert.Contains(t, foundContents, "systemctl enable chrony-wait")
}
