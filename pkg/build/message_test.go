package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureMessage(t *testing.T) {
	// Setup
	context, err := NewContext("", "", true)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, CleanUpBuildDir(context))
	}()

	builder := Builder{context: context}

	// Test
	script, err := builder.configureMessage()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.context.CombustionDir, messageScriptName))
	require.NoError(t, err)

	assert.Equal(t, messageScriptName, script)
}
