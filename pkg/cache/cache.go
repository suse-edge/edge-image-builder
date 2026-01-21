package cache

import (
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"go.uber.org/zap"
)

type Cache struct {
	cacheDir string
	enabled  bool
}

func New(cacheDir string) (*Cache, error) {
	if cacheDir == "" {
		return &Cache{cacheDir: cacheDir, enabled: false}, nil
	}

	return &Cache{cacheDir: cacheDir, enabled: true}, nil
}

func (cache *Cache) CacheEnabled() bool {
	return cache.enabled
}

func (cache *Cache) Get(fileIdentifier string) (path string, err error) {
	if !cache.CacheEnabled() {
		return "", nil
	}

	path, err = cache.identifierPath(fileIdentifier)
	if err != nil {
		return "", fmt.Errorf("searching for identifier '%s' in cache: %w", fileIdentifier, err)
	}

	if !exists(path) {
		return "", fs.ErrNotExist
	}

	return path, nil
}

func (cache *Cache) Put(fileIdentifier string, reader io.Reader) error {
	if !cache.CacheEnabled() {
		return nil
	}

	path, err := cache.identifierPath(fileIdentifier)
	if err != nil {
		return fmt.Errorf("searching for identifier '%s' in cache: %w", fileIdentifier, err)
	}

	if exists(path) {
		zap.S().Warnf("File with identifier '%s' already exists in cache", fileIdentifier)
		return fs.ErrExist
	}

	zap.S().Infof("Storing file with identifier '%s' in cache", fileIdentifier)

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, reader); err != nil {
		return fmt.Errorf("storing file: %w", err)
	}

	return nil
}

func (cache *Cache) identifierPath(fileIdentifier string) (string, error) {
	identifier, err := identifierHash(fileIdentifier)
	if err != nil {
		return "", err
	}

	return filepath.Join(cache.cacheDir, identifier), nil
}

func identifierHash(identifier string) (string, error) {
	h := fnv.New64()

	if _, err := h.Write([]byte(identifier)); err != nil {
		return "", err
	}

	hash := strconv.FormatUint(h.Sum64(), 10)

	zap.S().Debugf("Generated hash '%s' from identifier '%s'", hash, identifier)
	return hash, nil
}

func exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.S().Warnf("Looking for file with identifier failed: %v", err)
		}

		return false
	}

	return true
}
