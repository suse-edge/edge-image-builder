package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureMessage(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	scripts, err := configureMessage(ctx)

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(ctx.CombustionDir, messageScriptName))
	require.NoError(t, err)

	require.Len(t, scripts, 1)
	assert.Equal(t, messageScriptName, scripts[0])
}
