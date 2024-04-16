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

func HelmCharts(helm *image.Helm, valuesDir, buildDir, kubeVersion string, helmClient image.HelmClient) ([]*HelmChart, error) {
	var charts []*HelmChart
	chartRepoMap := mapChartRepos(helm)

	for _, helmChart := range helm.Charts {
		c := helmChart
		r, ok := chartRepoMap[c.RepositoryName]
		if !ok {
			return nil, fmt.Errorf("repository not found for chart %s", c.Name)
		}

		chart, err := handleChart(&c, r, valuesDir, buildDir, kubeVersion, helmClient)
		if err != nil {
			return nil, fmt.Errorf("handling chart resource: %w", err)
		}

		charts = append(charts, chart)
	}

	return charts, nil
}

func handleChart(chart *image.HelmChart, repo *image.HelmRepository, valuesDir, buildDir, kubeVersion string, helmClient image.HelmClient) (*HelmChart, error) {
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

	chartPath, err := downloadChart(chart, repo, helmClient, buildDir)
	if err != nil {
		return nil, fmt.Errorf("downloading chart: %w", err)
	}

	images, err := getChartContainerImages(chart, helmClient, chartPath, valuesPath, kubeVersion)
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

func downloadChart(chart *image.HelmChart, repo *image.HelmRepository, helmClient image.HelmClient, destDir string) (string, error) {
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

func getChartContent(chartPath string) (string, error) {
	data, err := os.ReadFile(chartPath)
	if err != nil {
		return "", fmt.Errorf("reading chart: %w", err)
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func getChartContainerImages(chart *image.HelmChart, helmClient image.HelmClient, chartPath, valuesPath, kubeVersion string) ([]string, error) {
	chartResources, err := helmClient.Template(chart.Name, chartPath, chart.Version, valuesPath, kubeVersion, chart.TargetNamespace)
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
