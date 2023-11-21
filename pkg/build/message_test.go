package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/context"
)

func TestConfigureMessage(t *testing.T) {
	// Setup
	ctx, err := context.NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, context.CleanUpBuildDir(ctx))
	}()

	builder := Builder{context: ctx}

	// Test
	script, err := builder.configureMessage()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.context.CombustionDir, messageScriptName))
	require.NoError(t, err)

	assert.Equal(t, messageScriptName, script)
}
