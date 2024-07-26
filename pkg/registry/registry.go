package registry

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/http"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"go.uber.org/zap"
)

type helmClient interface {
	AddRepo(repository *image.HelmRepository) error
	RegistryLogin(repository *image.HelmRepository) error
	Pull(chart string, repository *image.HelmRepository, version, destDir string) (string, error)
	Template(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string, apiVersions []string) ([]map[string]any, error)
}

type helmChart struct {
	image.HelmChart
	localPath     string
	repositoryURL string
}

type Registry struct {
	embeddedImages []image.ContainerImage
	manifestsDir   string
	helmClient     helmClient
	helmCharts     []*helmChart
	helmValuesDir  string
	kubeVersion    string
}

func New(ctx *image.Context, localManifestsDir string, helmClient helmClient, helmValuesDir string) (*Registry, error) {
	manifestsDir, err := storeManifests(ctx, localManifestsDir)
	if err != nil {
		return nil, fmt.Errorf("storing manifests: %w", err)
	}

	charts, err := storeHelmCharts(ctx, helmClient)
	if err != nil {
		return nil, fmt.Errorf("storing helm charts: %w", err)
	}

	return &Registry{
		embeddedImages: ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages,
		manifestsDir:   manifestsDir,
		helmClient:     helmClient,
		helmCharts:     charts,
		helmValuesDir:  helmValuesDir,
		kubeVersion:    ctx.ImageDefinition.Kubernetes.Version,
	}, nil
}

func (r *Registry) ManifestsPath() string {
	return r.manifestsDir
}

func storeManifests(ctx *image.Context, localManifestsDir string) (string, error) {
	const manifestsDir = "manifests"

	var manifestsPathPopulated bool

	manifestsDestDir := filepath.Join(ctx.BuildDir, manifestsDir)

	manifestURLs := ctx.ImageDefinition.Kubernetes.Manifests.URLs
	if len(manifestURLs) != 0 {
		if err := os.MkdirAll(manifestsDestDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("creating manifests dir: %w", err)
		}

		for index, manifestURL := range manifestURLs {
			filePath := filepath.Join(manifestsDestDir, fmt.Sprintf("dl-manifest-%d.yaml", index+1))

			if err := http.DownloadFile(context.Background(), manifestURL, filePath, nil); err != nil {
				return "", fmt.Errorf("downloading manifest '%s': %w", manifestURL, err)
			}
		}

		manifestsPathPopulated = true
	}

	if _, err := os.Stat(localManifestsDir); err == nil {
		if err = fileio.CopyFiles(localManifestsDir, manifestsDestDir, "", false); err != nil {
			return "", fmt.Errorf("copying manifests: %w", err)
		}

		manifestsPathPopulated = true
	} else if !errors.Is(err, fs.ErrNotExist) {
		zap.S().Warnf("Searching for local manifests failed: %v", err)
	}

	if !manifestsPathPopulated {
		return "", nil
	}

	return manifestsDestDir, nil
}

func storeHelmCharts(ctx *image.Context, helmClient helmClient) ([]*helmChart, error) {
	helm := &ctx.ImageDefinition.Kubernetes.Helm

	if len(helm.Charts) == 0 {
		return nil, nil
	}

	bar := progressbar.Default(int64(len(helm.Charts)), "Pulling selected Helm charts...")

	helmDir := filepath.Join(ctx.BuildDir, "helm")
	if err := os.MkdirAll(helmDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating helm directory: %w", err)
	}

	chartRepositories := mapChartsToRepos(helm)

	var charts []*helmChart

	for _, chart := range helm.Charts {
		c := chart
		repository, ok := chartRepositories[c.RepositoryName]
		if !ok {
			return nil, fmt.Errorf("repository not found for chart %s", c.Name)
		}

		localPath, err := downloadChart(helmClient, &c, repository, helmDir)
		if err != nil {
			return nil, fmt.Errorf("downloading chart: %w", err)
		}

		charts = append(charts, &helmChart{
			HelmChart:     c,
			localPath:     localPath,
			repositoryURL: repository.URL,
		})

		_ = bar.Add(1)
	}

	return charts, nil
}

func mapChartsToRepos(helm *image.Helm) map[string]*image.HelmRepository {
	chartRepoMap := make(map[string]*image.HelmRepository)

	for _, chart := range helm.Charts {
		for _, repo := range helm.Repositories {
			if chart.RepositoryName == repo.Name {
				r := repo
				chartRepoMap[chart.RepositoryName] = &r
			}
		}
	}

	return chartRepoMap
}

func downloadChart(helmClient helmClient, chart *image.HelmChart, repo *image.HelmRepository, destDir string) (string, error) {
	if strings.HasPrefix(repo.URL, "http") {
		if err := helmClient.AddRepo(repo); err != nil {
			return "", fmt.Errorf("adding repo: %w", err)
		}
	} else if repo.Authentication.Username != "" && repo.Authentication.Password != "" {
		if err := helmClient.RegistryLogin(repo); err != nil {
			return "", fmt.Errorf("logging into registry: %w", err)
		}
	}

	chartPath, err := helmClient.Pull(chart.Name, repo, chart.Version, destDir)
	if err != nil {
		return "", fmt.Errorf("pulling chart: %w", err)
	}

	return chartPath, nil
}

func (r *Registry) ContainerImages() ([]string, error) {
	manifestImages, err := r.manifestImages()
	if err != nil {
		return nil, fmt.Errorf("getting container images from manifests: %w", err)
	}

	chartImages, err := r.helmChartImages()
	if err != nil {
		return nil, fmt.Errorf("getting container images from helm charts: %w", err)
	}

	return deduplicateContainerImages(r.embeddedImages, manifestImages, chartImages), nil
}

func deduplicateContainerImages(embeddedImages []image.ContainerImage, manifestImages, chartImages []string) []string {
	imageSet := map[string]bool{}

	for _, img := range embeddedImages {
		imageSet[img.Name] = true
	}

	for _, img := range manifestImages {
		imageSet[img] = true
	}

	for _, img := range chartImages {
		imageSet[img] = true
	}

	var images []string

	for img := range imageSet {
		images = append(images, img)
	}

	return images
}
