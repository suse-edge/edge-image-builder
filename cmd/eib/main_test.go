package main

import (
	"flag"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestHelp(t *testing.T) {
	// Test
	args := []string{"-help"}
	imageConfig, _, err := parseFlags("eib", args)

	// Verify
	require.Nil(t, imageConfig)
	require.Error(t, err)
	require.Equal(t, flag.ErrHelp, err)
}

func TestVerbose(t *testing.T) {
	// Test
	args := []string{"-" + argVerbose}
	parseFlags("eib", args)

	// Verify
	require.Equal(t, log.DebugLevel, log.GetLevel())
}

func TestConfigFileNotSpecified(t *testing.T) {
	// Test
	imageConfig, _, err := parseFlags("eib", nil)

	// Verify
	require.Nil(t, imageConfig)
	require.Error(t, err)
	require.ErrorContains(t, err, "must be specified")
}

func TestConfigFileDoesntExist(t *testing.T) {
	// Test
	args := []string{"-" + argConfigFile, "fakefile"}
	imageConfig, _, err := parseFlags("eib", args)

	// Verify
	require.Nil(t, imageConfig)
	require.Error(t, err)
	require.ErrorContains(t, err, "does not exist")
}

func TestConfigFileAsDir(t *testing.T) {
	// Test
	args := []string{"-" + argConfigFile, "."}
	imageConfig, _, err := parseFlags("eib", args)

	// Verify
	require.Nil(t, imageConfig)
	require.Error(t, err)
	require.ErrorContains(t, err, "cannot be a directory")
}

func TestConfigFileParseFails(t *testing.T) {
	// Test
	args := []string{"-" + argConfigFile, "main_test.go"}
	imageConfig, _, err := parseFlags("eib", args)

	// Verify
	require.Nil(t, imageConfig)
	require.Error(t, err)
	require.ErrorContains(t, err, "error parsing")
}
