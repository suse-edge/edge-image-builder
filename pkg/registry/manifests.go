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
	containerImages := make(map[string]bool)

	entries, err := os.ReadDir(r.manifestsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading manifest dir: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(r.manifestsDir, entry.Name())

		resources, err := readManifest(path)
		if err != nil {
			return nil, fmt.Errorf("reading manifest '%s': %w", path, err)
		}

		for _, resource := range resources {
			extractManifestImages(resource, containerImages)
		}
	}

	var images []string

	for imageName := range containerImages {
		images = append(images, imageName)
	}

	return images, nil
}

func readManifest(manifestPath string) ([]map[string]any, error) {
	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("opening manifest: %w", err)
	}
	defer manifestFile.Close()

	var resources []map[string]any

	decoder := yaml.NewDecoder(manifestFile)
	for {
		var r map[string]any

		if err = decoder.Decode(&r); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("unmarshalling manifest: %w", err)
		}

		resources = append(resources, r)
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("invalid manifest")
	}

	return resources, nil
}

func extractManifestImages(resource map[string]any, images map[string]bool) {
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
