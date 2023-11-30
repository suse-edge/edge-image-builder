package network

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (inputDir, outputDir string, teardown func()) {
	inputConfigDir, err := os.MkdirTemp("", "eib-network-config-input-")
	require.NoError(t, err)

	outputConfigDir, err := os.MkdirTemp("", "eib-network-config-output-")
	require.NoError(t, err)

	return inputConfigDir, outputConfigDir, func() {
		assert.NoError(t, os.RemoveAll(outputConfigDir))
		assert.NoError(t, os.RemoveAll(inputConfigDir))
	}
}

func TestGenerateCommand(t *testing.T) {
	inputConfigDir, outputConfigDir, teardown := setup(t)
	defer teardown()

	var sb strings.Builder

	cmd := generateCommand(inputConfigDir, outputConfigDir, &sb)

	expectedArgs := []string{
		"nmc",
		"generate",
		"--config-dir", inputConfigDir,
		"--output-dir", outputConfigDir,
	}

	assert.Equal(t, expectedArgs, cmd.Args)
	assert.Equal(t, &sb, cmd.Stdout)
	assert.Equal(t, &sb, cmd.Stderr)
}

// TODO: Set up working example once nmc is available as an RPM
func TestConfigGenerator_GenerateNetworkConfig_MissingExecutable(t *testing.T) {
	inputConfigDir, outputConfigDir, teardown := setup(t)
	defer teardown()

	var sb strings.Builder
	var generator ConfigGenerator

	err := generator.GenerateNetworkConfig(inputConfigDir, outputConfigDir, &sb)
	require.Error(t, err)
	assert.ErrorContains(t, err, "running generate command")
	assert.ErrorContains(t, err, "executable file not found")
}
