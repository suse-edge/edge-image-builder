package registry

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

func (r *Registry) manifestImages() ([]string, error) {
	var imageSet = make(map[string]bool)

	entries, err := os.ReadDir(r.manifestsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading manifest dir: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(r.manifestsDir, entry.Name())

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
