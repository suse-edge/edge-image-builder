package combustion

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
	"github.com/suse-edge/edge-image-builder/pkg/context"
)

func setupContext(t *testing.T) (ctx *context.Context, teardown func()) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	buildDir, err := os.MkdirTemp("", "eib-build-")
	require.NoError(t, err)

	combustionDir, err := os.MkdirTemp("", "eib-combustion-")
	require.NoError(t, err)

	ctx = &context.Context{
		ImageConfigDir: configDir,
		BuildDir:       buildDir,
		CombustionDir:  combustionDir,
		ImageConfig:    &config.ImageConfig{},
	}

	return ctx, func() {
		assert.NoError(t, os.RemoveAll(combustionDir))
		assert.NoError(t, os.RemoveAll(buildDir))
		assert.NoError(t, os.RemoveAll(configDir))
	}
}
