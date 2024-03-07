package registry

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type HelmChart struct {
	CRD             HelmCRD
	ContainerImages []string
}

func HelmCharts(helmCharts []image.HelmChart, valuesDir, buildDir, kubeVersion string, helm image.Helm) ([]*HelmChart, error) {
	var charts []*HelmChart

	for _, helmChart := range helmCharts {
		c := helmChart
		chart, err := handleChart(&c, valuesDir, buildDir, kubeVersion, helm)
		if err != nil {
			return nil, fmt.Errorf("handling chart resource: %w", err)
		}

		charts = append(charts, chart)
	}

	return charts, nil
}

func handleChart(chart *image.HelmChart, valuesDir, buildDir, kubeVersion string, helm image.Helm) (*HelmChart, error) {
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

	chartPath, err := downloadChart(chart, helm, buildDir)
	if err != nil {
		return nil, fmt.Errorf("downloading chart: %w", err)
	}

	chartResources, err := helm.Template(chart.Name, chartPath, chart.Version, valuesPath, kubeVersion, nil)
	if err != nil {
		return nil, fmt.Errorf("templating chart: %w", err)
	}

	containerImages := map[string]bool{}
	for _, chartResource := range chartResources {
		storeManifestImages(chartResource, containerImages)
	}

	var images []string
	for i := range containerImages {
		images = append(images, i)
	}

	chartContent, err := getChartContent(chartPath)
	if err != nil {
		return nil, fmt.Errorf("getting chart content: %w", err)
	}

	helmChart := HelmChart{
		CRD:             NewHelmCRD(chart, chartContent, string(valuesContent)),
		ContainerImages: images,
	}

	return &helmChart, nil
}

func downloadChart(chart *image.HelmChart, helm image.Helm, destDir string) (string, error) {
	if err := helm.AddRepo(chart.Name, chart.Repo); err != nil {
		return "", fmt.Errorf("adding repo: %w", err)
	}

	chartPath, err := helm.Pull(chart.Name, chart.Repo, chart.Version, destDir)
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
