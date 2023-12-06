package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func setupElementalConfigDir(t *testing.T) (ctx *image.Context, teardown func()) {
	ctx, teardownCtx := setupContext(t)

	testConfigDir := filepath.Join(ctx.ImageConfigDir, elementalConfigDir)
	err := os.Mkdir(testConfigDir, 0o755)
	require.NoError(t, err)

	testConfigFile := filepath.Join(testConfigDir, elementalConfigName)
	contents := "foo: bar"
	err = os.WriteFile(testConfigFile, []byte(contents), 0o600)
	require.NoError(t, err)

	teardown = teardownCtx

	return
}

func TestCopyElementalConfigFile(t *testing.T) {
	// Setup
	ctx, teardown := setupElementalConfigDir(t)
	defer teardown()

	// Test
	err := copyElementalConfigFile(ctx)

	// Verify
	require.NoError(t, err)

	foundFile := filepath.Join(ctx.CombustionDir, elementalConfigName)
	found, err := os.ReadFile(foundFile)
	require.NoError(t, err)
	assert.Equal(t, "foo: bar", string(found))
}

func TestWriteElementalCombustionScript(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	err := writeElementalCombustionScript(ctx)

	// Verify
	require.NoError(t, err)

	scriptFilename := filepath.Join(ctx.CombustionDir, elementalScriptName)
	_, err = os.Stat(scriptFilename)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(scriptFilename)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, "elemental-register --config-path /etc/elemental/config.yaml")
}
