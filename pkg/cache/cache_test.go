package cache

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	defaultCacheDir       = "test-cache"
	defaultFileIdentifier = "some-cool-filename"
	defaultFileContents   = "some-data"
)

func setup(t *testing.T, cacheDir string) (cache *Cache, teardown func()) {
	if cacheDir == "" {
		return nil, func() {}
	}

	assert.NoError(t, os.MkdirAll(cacheDir, os.ModePerm))

	cache, err := New(cacheDir)
	require.NoError(t, err)

	return cache, func() {
		assert.NoError(t, os.RemoveAll("test-cache"))
	}
}

func TestCache(t *testing.T) {
	cache, teardown := setup(t, defaultCacheDir)
	defer teardown()

	fileIdentifier := defaultFileIdentifier
	fileContents := defaultFileContents

	require.NoError(t, cache.Put(fileIdentifier, strings.NewReader(fileContents)))

	path, err := cache.Get(fileIdentifier)
	require.NoError(t, err)
	require.FileExists(t, path)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, fileContents, string(b))

	assert.NotNil(t, cache)
}

func TestCacheNoDir(t *testing.T) {
	cache, teardown := setup(t, "")
	defer teardown()

	fileIdentifier := defaultFileIdentifier
	fileContents := defaultFileContents

	// No error because the Put function immediately returns nil when cache is disabled
	require.NoError(t, cache.Put(fileIdentifier, strings.NewReader(fileContents)))

	// No error because the Get function immediately returns nil when cache is disabled
	// But we confirm that the File doesn't exist
	path, err := cache.Get(fileIdentifier)
	require.NoError(t, err)
	require.NoFileExists(t, path)

	assert.Nil(t, cache)
}

func TestCache_MissingEntry(t *testing.T) {
	cache, teardown := setup(t, defaultCacheDir)
	defer teardown()

	fileIdentifier := defaultFileIdentifier

	path, err := cache.Get(fileIdentifier)
	require.Error(t, err)

	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.NoFileExists(t, path)
}

func TestCache_DoubleInsert(t *testing.T) {
	cache, teardown := setup(t, defaultCacheDir)
	defer teardown()

	fileIdentifier := "https://raw.githubusercontent.com/suse-edge/edge-image-builder/main/README.md"
	fileContents := defaultFileContents

	require.NoError(t, cache.Put(fileIdentifier, strings.NewReader(fileContents)))
	assert.ErrorIs(t, cache.Put(fileIdentifier, strings.NewReader(fileContents)), fs.ErrExist)
}
