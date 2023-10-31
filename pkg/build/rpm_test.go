package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestGetRPMFileNames(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		ImageConfigDir: "../config/testdata",
	}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	// Test
	err = builder.getRPMFileNames()

	// Verify
	require.NoError(t, err)

	assert.Contains(t, builder.rpmFileNames, "rpm1.rpm")
	assert.Contains(t, builder.rpmFileNames, "rpm2.rpm")
}

func TestCopyRPMs(t *testing.T) {
	// Setup
	bc := config.BuildConfig{
		ImageConfigDir: "../config/testdata",
	}
	builder := New(nil, &bc)
	err := builder.prepareBuildDir()
	require.NoError(t, err)
	defer os.Remove(builder.eibBuildDir)

	// Test
	err = builder.copyRPMs()

	// Verify
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, "rpm1.rpm"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(builder.combustionDir, "rpm2.rpm"))
	require.NoError(t, err)
}
