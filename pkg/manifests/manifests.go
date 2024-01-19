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

func configureManifests(ctx *image.Context) error {
	var downloadedManifests []string
	var combinedManifestPaths []string
	var extractedImages []string
	var err error // created here to avoid scoping issues with "downloadedManifests" when using := on line 29

	if len(ctx.ImageDefinition.Kubernetes.Manifests.URLs) != 0 {
		downloadDestination := filepath.Join(ctx.CombustionDir, "downloaded-manifests")
		if err := os.MkdirAll(downloadDestination, os.ModePerm); err != nil {
			return fmt.Errorf("creating %s dir: %w", downloadDestination, err)
		}

		downloadedManifests, err = downloadManifests(ctx, downloadDestination)
		if err != nil {
			return fmt.Errorf("error downloading manifests: %w", err)
		}
	}

	localManifestSrcDir := filepath.Join(ctx.ImageConfigDir, "manifests")
	localManifestDestDir := filepath.Join(ctx.ImageConfigDir, "local-manifests")
	copiedManifests, err := copyManifests(localManifestSrcDir, localManifestDestDir)
	if err != nil {
		return fmt.Errorf("error copying manifests: %w", err)
	}

	combinedManifestPaths = append(copiedManifests, downloadedManifests...)

	for _, manifestPath := range combinedManifestPaths {
		manifestData, err := readManifest(manifestPath)
		if err != nil {
			return fmt.Errorf("error reading manifest %w", err)
		}

		foundImages, err := findImagesInManifest(manifestData)
		if err != nil {
			return fmt.Errorf("error finding images in manifest %w", err)
		}
		extractedImages = append(extractedImages, foundImages...)
	}

	extractedImages = removeDuplicateImages(extractedImages)
	addImagesToRegistryDefinition(ctx, extractedImages)

	return nil
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

	var manifestPaths []string

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
		manifestPaths = append(manifestPaths, destPath)

	}

	return manifestPaths, nil
}

func downloadManifests(ctx *image.Context, destPath string) ([]string, error) {
	manifests := ctx.ImageDefinition.Kubernetes.Manifests.URLs
	var manifestPaths []string

	for index, manifestURL := range manifests {
		filePath := filepath.Join(destPath, fmt.Sprintf("manifest-%d.yaml", index+1))
		manifestPaths = append(manifestPaths, filePath)

		if err := http.DownloadFile(context.Background(), manifestURL, filePath); err != nil {
			return nil, fmt.Errorf("downloading manifest '%s': %w", manifestURL, err)
		}
	}

	return manifestPaths, nil
}

func removeDuplicateImages(images []string) []string {
	imagesSet := make(map[string]bool)
	var formattedImages []string
	for _, item := range images {
		if _, value := imagesSet[item]; !value {
			imagesSet[item] = true
			formattedImages = append(formattedImages, item)
		}
	}

	return formattedImages
}

func addImagesToRegistryDefinition(ctx *image.Context, manifestImages []string) {
	for _, imageName := range manifestImages {
		if !isInRegistry(ctx, imageName) {
			containerImageDef := image.ContainerImage{
				Name: imageName,
			}
			ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages = append(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages, containerImageDef)
		}
	}
}

func isInRegistry(ctx *image.Context, imageName string) bool {
	for _, containerImage := range ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages {
		if imageName == containerImage.Name {
			return true
		}
	}

	return false
}
