//go:build integration

package http

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile_Successful_NoCache(t *testing.T) {
	url := "https://raw.githubusercontent.com/suse-edge/edge-image-builder/main/README.md"
	path := "README.md"

	require.NoError(t, DownloadFile(context.Background(), url, path, nil))
	defer func() {
		assert.NoError(t, os.Remove(path))
	}()

	assert.FileExists(t, path)
}

func TestDownloadFile_Successful_Cache(t *testing.T) {
	url := "https://raw.githubusercontent.com/suse-edge/edge-image-builder/main/README.md"
	path := "README.md"

	var sb strings.Builder

	require.NoError(t, DownloadFile(context.Background(), url, path, &sb))
	defer func() {
		assert.NoError(t, os.Remove(path))
	}()

	require.FileExists(t, path)

	b, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.Equal(t, sb.String(), string(b))
}
