package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func TestCopyCombustionFile(t *testing.T) {
	// Setup
	bc := config.BuildConfig{}
	err := prepareBuildDir(&bc)
	require.NoError(t, err)
	defer os.Remove(bc.BuildTempDir)

	filename := filepath.Join(bc.BuildTempDir, "eib-test")
	testData := []byte("EIB")
	err = os.WriteFile(filename, testData, os.ModePerm)
	require.NoError(t, err)
	defer os.Remove(filename)

	// Test
	err = copyCombustionFile(filename, &bc)

	// Verify
	require.NoError(t, err)
	foundData, err := os.ReadFile(filepath.Join(bc.CombustionDir, "eib-test"))
	require.NoError(t, err)
	require.Equal(t, testData, foundData)
}
