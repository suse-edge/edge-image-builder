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

func TestConfigureProxy_NoConf(t *testing.T) {
	// Setup
	var ctx image.Context

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Proxy: image.Proxy{},
		},
	}

	// Test
	scripts, err := configureProxy(&ctx)

	// Verify
	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureProxy_FullConfiguration(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		OperatingSystem: image.OperatingSystem{
			Proxy: image.Proxy{
				HTTPProxy:  "http://10.0.0.1:3128",
				HTTPSProxy: "http://10.0.0.1:3128",
				NoProxy:    []string{"localhost", "127.0.0.1", "edge.suse.com"},
			},
		},
	}

	// Test
	scripts, err := configureProxy(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, proxyScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, proxyScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)

	// - Make sure that the global PROXY_ENABLED="yes" flag is set because either http/https proxy is set
	assert.Contains(t, foundContents, "s|PROXY_ENABLED=.*|PROXY_ENABLED=\"yes\"|g", "global proxy has not been enabled")

	// - Ensure that we have the HTTP_PROXY set correctly
	assert.Contains(t, foundContents, "s|HTTP_PROXY=.*|HTTP_PROXY=\"http://10.0.0.1:3128\"|g", "HTTP_PROXY not set correctly")

	// - Ensure that we have the HTTPS_PROXY set correctly
	assert.Contains(t, foundContents, "s|HTTPS_PROXY=.*|HTTPS_PROXY=\"http://10.0.0.1:3128\"|g", "HTTPS_PROXY not set correctly")

	// - Ensure that we have the NO_PROXY list overridden
	assert.Contains(t, foundContents, "s|NO_PROXY=.*|NO_PROXY=\"localhost, 127.0.0.1, edge.suse.com\"|g", "NO_PROXY not set correctly")
}
