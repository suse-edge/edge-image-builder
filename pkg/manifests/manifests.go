package manifests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/http"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func configureManifests() {

}

func readManifest(manifestPath string) (interface{}, error) {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("error reading manifest: %w", err)
	}

	if len(manifestData) == 0 {
		return nil, fmt.Errorf("invalid manifest")
	}

	var manifest interface{}
	err = yaml.Unmarshal(manifestData, &manifest)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling manifest YAML: %w", err)
	}
	return manifest, nil
}

func findImagesInManifest(data interface{}) ([]string, error) {
	imageSet := make(map[string]struct{})

	var findImages func(data interface{})
	findImages = func(data interface{}) {
		switch t := data.(type) {
		case map[string]interface{}:
			for k, v := range t {
				if k == "image" {
					if imageName, ok := v.(string); ok {
						imageSet[imageName] = struct{}{}
					}
				}
				findImages(v)
			}
		case []interface{}:
			for _, v := range t {
				findImages(v)
			}
		}
	}

	findImages(data)

	images := make([]string, 0, len(imageSet))
	for uniqueImage := range imageSet {
		images = append(images, uniqueImage)
	}

	return images, nil
}

func copyManifests(src string, dest string) ([]string, error) {
	if dest == "" {
		return nil, fmt.Errorf("manifest destination directory not defined")
	}

	var list []string

	manifests, err := os.ReadDir(src)
	if err != nil {
		return nil, fmt.Errorf("reading manifest source dir: %w", err)
	}

	for _, manifest := range manifests {
		manifestName := strings.ToLower(manifest.Name())
		if filepath.Ext(manifestName) != ".yaml" && filepath.Ext(manifestName) != ".yml" {
			zap.S().Warnf("Skipping %s as it is not a yaml file", manifest.Name())
			continue
		}

		sourcePath := filepath.Join(src, manifest.Name())
		destPath := filepath.Join(dest, manifest.Name())
		err := fileio.CopyFile(sourcePath, destPath, fileio.NonExecutablePerms)
		if err != nil {
			return nil, fmt.Errorf("copying manifest file %s: %w", sourcePath, err)
		}
		list = append(list, manifest.Name())

	}

	return list, nil
}

func downloadManifests(ctx image.Context, destPath string) error {
	manifestURLs := ctx.ImageDefinition.Kubernetes.Manifests.URLs

	for _, manifestURL := range manifestURLs {
		if err := http.DownloadFile(context.Background(), manifestURL, destPath); err != nil {
			return fmt.Errorf("downloading manifest '%s': %w", manifestURL, err)
		}
	}

	return nil
}
