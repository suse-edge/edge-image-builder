package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/http"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func ManifestImages(manifestURLs []string, manifestsDir string) ([]string, error) {
	var manifestPaths []string

	if len(manifestURLs) != 0 {
		paths, err := DownloadManifests(manifestURLs, os.TempDir())
		if err != nil {
			return nil, fmt.Errorf("downloading manifests: %w", err)
		}

		manifestPaths = append(manifestPaths, paths...)
	}

	if manifestsDir != "" {
		paths, err := getManifestPaths(manifestsDir)
		if err != nil {
			return nil, fmt.Errorf("getting local manifest paths: %w", err)
		}

		manifestPaths = append(manifestPaths, paths...)
	}

	var imageSet = make(map[string]bool)

	for _, path := range manifestPaths {
		manifests, err := readManifest(path)
		if err != nil {
			return nil, fmt.Errorf("reading manifest: %w", err)
		}

		for _, manifestData := range manifests {
			storeManifestImages(manifestData, imageSet)
		}
	}

	var images []string

	for imageName := range imageSet {
		images = append(images, imageName)
	}

	return images, nil
}

func readManifest(manifestPath string) ([]map[string]any, error) {
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("error opening manifest: %w", err)
	}

	var manifests []map[string]any
	decoder := yaml.NewDecoder(manifestFile)
	for {
		var manifest map[string]any
		err = decoder.Decode(&manifest)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling manifest yaml '%s': %w", manifestPath, err)
		}
		manifests = append(manifests, manifest)
	}

	if len(manifests) == 0 {
		return nil, fmt.Errorf("invalid manifest")
	}

	return manifests, nil
}

func storeManifestImages(resource map[string]any, images map[string]bool) {
	var k8sKinds = []string{
		"Pod",
		"Deployment",
		"StatefulSet",
		"DaemonSet",
		"ReplicaSet",
		"Job",
		"CronJob",
	}

	kind, _ := resource["kind"].(string)
	if !slices.Contains(k8sKinds, kind) {
		return
	}

	var findImages func(data any)

	findImages = func(data any) {
		switch t := data.(type) {
		case map[string]any:
			for k, v := range t {
				if k == "image" {
					if imageName, ok := v.(string); ok {
						images[imageName] = true
					}
				}
				findImages(v)
			}
		case []any:
			for _, v := range t {
				findImages(v)
			}
		}
	}

	findImages(resource)
}

func getManifestPaths(src string) ([]string, error) {
	if src == "" {
		return nil, fmt.Errorf("manifest source directory not defined")
	}

	var manifestPaths []string

	manifests, err := os.ReadDir(src)
	if err != nil {
		return nil, fmt.Errorf("reading manifest source dir '%s': %w", src, err)
	}

	for _, manifest := range manifests {
		manifestName := strings.ToLower(manifest.Name())
		if filepath.Ext(manifestName) != ".yaml" && filepath.Ext(manifestName) != ".yml" {
			zap.S().Warnf("Skipping %s as it is not a yaml file", manifest.Name())
			continue
		}

		sourcePath := filepath.Join(src, manifest.Name())
		manifestPaths = append(manifestPaths, sourcePath)
	}

	return manifestPaths, nil
}

func DownloadManifests(manifestURLs []string, destPath string) ([]string, error) {
	var manifestPaths []string

	for index, manifestURL := range manifestURLs {
		filePath := filepath.Join(destPath, fmt.Sprintf("dl-manifest-%d.yaml", index+1))
		manifestPaths = append(manifestPaths, filePath)

		if err := http.DownloadFile(context.Background(), manifestURL, filePath, nil); err != nil {
			return nil, fmt.Errorf("downloading manifest '%s': %w", manifestURL, err)
		}
	}

	return manifestPaths, nil
}
