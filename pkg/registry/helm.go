package registry

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func (r *Registry) HelmCharts() ([]*HelmCRD, error) {
	var crds []*HelmCRD

	for _, chart := range r.helmCharts {
		data, err := os.ReadFile(chart.localPath)
		if err != nil {
			return nil, fmt.Errorf("reading chart: %w", err)
		}

		chartContent := base64.StdEncoding.EncodeToString(data)

		var valuesContent []byte
		if chart.ValuesFile != "" {
			valuesPath := filepath.Join(r.helmValuesDir, chart.ValuesFile)
			valuesContent, err = os.ReadFile(valuesPath)
			if err != nil {
				return nil, fmt.Errorf("reading values content: %w", err)
			}
		}

		crd := NewHelmCRD(&chart.HelmChart, chartContent, string(valuesContent), chart.repositoryURL)
		crds = append(crds, crd)
	}

	return crds, nil
}

func (r *Registry) helmChartImages() ([]string, error) {
	var containerImages []string

	for _, chart := range r.helmCharts {
		var valuesPath string
		if chart.ValuesFile != "" {
			valuesPath = filepath.Join(r.helmValuesDir, chart.ValuesFile)
		}

		images, err := r.getChartContainerImages(&chart.HelmChart, chart.localPath, valuesPath, r.kubeVersion)
		if err != nil {
			return nil, err
		}

		containerImages = append(containerImages, images...)
	}

	return containerImages, nil
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
