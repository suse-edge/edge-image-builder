package registry

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"gopkg.in/yaml.v3"
)

type Helm interface {
	AddRepo(chart, repository string) error
	Pull(chart, repository, version, destDir string) (string, error)
	Template(chart, repository, version, valuesFilePath string, setArgs []string) ([]map[string]any, error)
}

type HelmChart struct {
	Filename        string
	Resources       []map[string]any
	ContainerImages []string
}

func HelmCharts(srcDir, buildDir string, helm Helm) ([]*HelmChart, error) {
	manifestPaths, err := getManifestPaths(srcDir)
	if err != nil {
		return nil, fmt.Errorf("getting helm manifest paths: %w", err)
	}

	var charts []*HelmChart

	for _, manifest := range manifestPaths {
		resources, err := parseManifest(manifest)
		if err != nil {
			return nil, fmt.Errorf("parsing manifest: %w", err)
		}

		containerImages := make(map[string]bool)
		chart := &HelmChart{
			Filename: filepath.Base(manifest),
		}

		for _, resource := range resources {
			kind, ok := resource["kind"].(string)
			if !ok {
				return nil, fmt.Errorf("resource is missing 'kind' field")
			}

			if kind == helmChartKind {
				if err = handleChartResource(resource, buildDir, helm, containerImages); err != nil {
					return nil, fmt.Errorf("handling chart resource: %w", err)
				}
			} else {
				storeManifestImages(resource, containerImages)
			}

			chart.Resources = append(chart.Resources, resource)
		}

		for image := range containerImages {
			chart.ContainerImages = append(chart.ContainerImages, image)
		}

		charts = append(charts, chart)
	}

	return charts, nil
}

func parseManifest(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var resources []map[string]any

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var r map[string]any
		if err = decoder.Decode(&r); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("unmarshaling resource: %w", err)
		}

		resources = append(resources, r)
	}

	return resources, nil
}

func handleChartResource(resource map[string]any, buildDir string, helm Helm, containerImages map[string]bool) error {
	crd, err := parseHelmCRD(resource)
	if err != nil {
		return fmt.Errorf("parsing crd: %w", err)
	}

	chartPath, err := downloadChart(crd, helm, buildDir)
	if err != nil {
		return fmt.Errorf("downloading chart: %w", err)
	}

	var valuesPath string
	if crd.Spec.ValuesContent != "" {
		valuesPath = filepath.Join(buildDir, fmt.Sprintf("values-%s.yaml", crd.Spec.Chart))

		if err = os.WriteFile(valuesPath, []byte(crd.Spec.ValuesContent), fileio.NonExecutablePerms); err != nil {
			return fmt.Errorf("writing chart values file: %w", err)
		}
	}

	var chartName string
	var modifyContent bool

	if crd.Spec.ChartContent == "" {
		chartName = crd.Spec.Chart
		modifyContent = true
	} else {
		chartName = crd.Metadata.Name
	}

	chartResources, err := helm.Template(chartName, chartPath, crd.Spec.Version, valuesPath, crd.parseSetArgs())
	if err != nil {
		return fmt.Errorf("templating chart: %w", err)
	}

	for _, chartResource := range chartResources {
		storeManifestImages(chartResource, containerImages)
	}

	if modifyContent {
		if err = setChartContent(resource, chartPath); err != nil {
			return fmt.Errorf("setting chart content: %w", err)
		}
	}

	return nil
}

func parseHelmCRD(resource map[string]any) (*helmCRD, error) {
	b, err := yaml.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshaling resource: %w", err)
	}

	var crd helmCRD
	if err = yaml.Unmarshal(b, &crd); err != nil {
		return nil, fmt.Errorf("unmarshaling CRD: %w", err)
	}

	return &crd, nil
}

func downloadChart(crd *helmCRD, helm Helm, destDir string) (string, error) {
	if crd.Spec.ChartContent == "" {
		if err := helm.AddRepo(crd.Spec.Chart, crd.Spec.Repo); err != nil {
			return "", fmt.Errorf("adding repo: %w", err)
		}

		chartPath, err := helm.Pull(crd.Spec.Chart, crd.Spec.Repo, crd.Spec.Version, destDir)
		if err != nil {
			return "", fmt.Errorf("pulling chart: %w", err)
		}

		return chartPath, nil
	}

	chartContents, err := base64.StdEncoding.DecodeString(crd.Spec.ChartContent)
	if err != nil {
		return "", fmt.Errorf("decoding base64 chart content: %w", err)
	}

	chartPath := filepath.Join(destDir, fmt.Sprintf("%s.tgz", crd.Metadata.Name))
	if err = os.WriteFile(chartPath, chartContents, fileio.NonExecutablePerms); err != nil {
		return "", fmt.Errorf("storing chart: %w", err)
	}

	return chartPath, nil
}

func setChartContent(resource map[string]any, chartPath string) error {
	spec, ok := resource["spec"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing 'spec' field")
	}

	if _, ok = spec["chartContent"].(string); ok {
		return fmt.Errorf("'chartContent' field is already set")
	}

	data, err := os.ReadFile(chartPath)
	if err != nil {
		return fmt.Errorf("reading chart: %w", err)
	}
	spec["chartContent"] = base64.StdEncoding.EncodeToString(data)

	delete(spec, "repo")
	delete(spec, "chart")

	return nil
}
