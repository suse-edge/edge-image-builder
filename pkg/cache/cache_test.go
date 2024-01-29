package cache

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (cache *Cache, teardown func()) {
	cache, err := New("test-cache")
	require.NoError(t, err)

	return cache, func() {
		assert.NoError(t, os.RemoveAll("test-cache"))
	}
}

func TestCache(t *testing.T) {
	cache, teardown := setup(t)
	defer teardown()

	fileIdentifier := "some-cool-filename"
	fileContents := "some-data"

	require.NoError(t, cache.Put(fileIdentifier, strings.NewReader(fileContents)))

	path, err := cache.Get(fileIdentifier)
	require.NoError(t, err)
	require.FileExists(t, path)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, fileContents, string(b))
}

func TestCache_MissingEntry(t *testing.T) {
	cache, teardown := setup(t)
	defer teardown()

	fileIdentifier := "some-cool-filename"

	path, err := cache.Get(fileIdentifier)
	require.Error(t, err)

	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.NoFileExists(t, path)
}

func TestCache_DoubleInsert(t *testing.T) {
	cache, teardown := setup(t)
	defer teardown()

	fileIdentifier := "https://raw.githubusercontent.com/suse-edge/edge-image-builder/main/README.md"
	fileContents := "some-data"

	require.NoError(t, cache.Put(fileIdentifier, strings.NewReader(fileContents)))
	assert.ErrorIs(t, cache.Put(fileIdentifier, strings.NewReader(fileContents)), fs.ErrExist)
}
