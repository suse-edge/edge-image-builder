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

func GetAllImagesAndCharts(ctx *image.Context) ([]image.ContainerImage, []image.HelmChart, error) {
	var downloadedManifestPaths []string
	var combinedManifestPaths []string
	var extractedImagesSet = make(map[string]string)
	var err error
	helmCharts := ctx.ImageDefinition.Kubernetes.HelmCharts

	if len(ctx.ImageDefinition.Kubernetes.Manifests.URLs) != 0 {
		downloadDestination := filepath.Join(ctx.BuildDir, "downloaded-manifests")
		if err = os.MkdirAll(downloadDestination, os.ModePerm); err != nil {
			return nil, nil, fmt.Errorf("creating %s dir: %w", downloadDestination, err)
		}

		downloadedManifestPaths, err = downloadManifests(ctx, downloadDestination)
		if err != nil {
			return nil, nil, fmt.Errorf("error downloading manifests: %w", err)
		}
	}

	localManifestSrcDir := filepath.Join(ctx.ImageConfigDir, "kubernetes", "manifests")
	localManifestPaths, err := getLocalManifestPaths(localManifestSrcDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting local manifest paths: %w", err)
	}

	combinedManifestPaths = append(localManifestPaths, downloadedManifestPaths...)

	for _, manifestPath := range combinedManifestPaths {
		manifestData, err := readManifest(manifestPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading manifest %w", err)
		}

		extractedImagesSet, err = findImagesInManifest(manifestData, extractedImagesSet)
		if err != nil {
			return nil, nil, fmt.Errorf("error finding images in manifest '%s': %w", manifestPath, err)
		}
	}

	for _, definedImage := range ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages {
		extractedImagesSet[definedImage.Name] = definedImage.SupplyChainKey
	}

	allImages := make([]image.ContainerImage, 0, len(extractedImagesSet))
	for uniqueImage := range extractedImagesSet {
		containerImage := image.ContainerImage{
			Name:           uniqueImage,
			SupplyChainKey: extractedImagesSet[uniqueImage],
		}
		allImages = append(allImages, containerImage)
	}

	return allImages, helmCharts, nil
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
		return nil, fmt.Errorf("error unmarshalling manifest yaml '%s': %w", manifestPath, err)
	}

	return manifest, nil
}

func findImagesInManifest(data interface{}, imageSet map[string]string) (map[string]string, error) {

	var findImages func(data interface{})
	findImages = func(data interface{}) {
		switch t := data.(type) {
		case map[string]interface{}:
			for k, v := range t {
				if k == "image" {
					if imageName, ok := v.(string); ok {
						imageSet[imageName] = ""
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

	return imageSet, nil
}

func getLocalManifestPaths(src string) ([]string, error) {
	if src == "" {
		return nil, fmt.Errorf("manifest source directory not defined")
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
		manifestPaths = append(manifestPaths, sourcePath)

	}

	return manifestPaths, nil
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
