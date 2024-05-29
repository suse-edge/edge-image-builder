package registry

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type HelmChart struct {
	CRD             HelmCRD
	ContainerImages []string
}

func (r *Registry) HelmCharts(helm *image.Helm, valuesDir, buildDir, kubeVersion string) ([]*HelmChart, error) {
	var charts []*HelmChart
	chartRepoMap := mapChartRepos(helm)

	for _, helmChart := range helm.Charts {
		chart := helmChart
		repository, ok := chartRepoMap[chart.RepositoryName]
		if !ok {
			return nil, fmt.Errorf("repository not found for chart %s", chart.Name)
		}

		c, err := r.handleChart(&chart, repository, valuesDir, buildDir, kubeVersion)
		if err != nil {
			return nil, fmt.Errorf("handling chart resource: %w", err)
		}

		charts = append(charts, c)
	}

	return charts, nil
}

func (r *Registry) handleChart(chart *image.HelmChart, repo *image.HelmRepository, valuesDir, buildDir, kubeVersion string) (*HelmChart, error) {
	var valuesPath string
	var valuesContent []byte
	if chart.ValuesFile != "" {
		var err error
		valuesPath = filepath.Join(valuesDir, chart.ValuesFile)
		valuesContent, err = os.ReadFile(valuesPath)
		if err != nil {
			return nil, fmt.Errorf("reading values content: %w", err)
		}
	}

	chartPath, err := r.downloadChart(chart, repo, buildDir)
	if err != nil {
		return nil, fmt.Errorf("downloading chart: %w", err)
	}

	images, err := r.getChartContainerImages(chart, chartPath, valuesPath, kubeVersion)
	if err != nil {
		return nil, fmt.Errorf("getting chart container images: %w", err)
	}

	chartContent, err := getChartContent(chartPath)
	if err != nil {
		return nil, fmt.Errorf("getting chart content: %w", err)
	}

	helmChart := HelmChart{
		CRD:             NewHelmCRD(chart, chartContent, string(valuesContent), repo.URL),
		ContainerImages: images,
	}

	return &helmChart, nil
}

func (r *Registry) downloadChart(chart *image.HelmChart, repo *image.HelmRepository, destDir string) (string, error) {
	if strings.HasPrefix(repo.URL, "http") {
		if err := r.helmClient.AddRepo(repo); err != nil {
			return "", fmt.Errorf("adding repo: %w", err)
		}
	} else if repo.Authentication.Username != "" && repo.Authentication.Password != "" {
		if err := r.helmClient.RegistryLogin(repo); err != nil {
			return "", fmt.Errorf("logging into registry: %w", err)
		}
	}

	chartPath, err := r.helmClient.Pull(chart.Name, repo, chart.Version, destDir)
	if err != nil {
		return "", fmt.Errorf("pulling chart: %w", err)
	}

	return chartPath, nil
}

func getChartContent(chartPath string) (string, error) {
	data, err := os.ReadFile(chartPath)
	if err != nil {
		return "", fmt.Errorf("reading chart: %w", err)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func (r *Registry) getChartContainerImages(chart *image.HelmChart, chartPath, valuesPath, kubeVersion string) ([]string, error) {
	chartResources, err := r.helmClient.Template(chart.Name, chartPath, chart.Version, valuesPath, kubeVersion, chart.TargetNamespace)
	if err != nil {
		return nil, fmt.Errorf("templating chart: %w", err)
	}

	containerImages := map[string]bool{}
	for _, resource := range chartResources {
		storeManifestImages(resource, containerImages)
	}

	var images []string
	for i := range containerImages {
		images = append(images, i)
	}

	return images, nil
}

func mapChartRepos(helm *image.Helm) map[string]*image.HelmRepository {
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
