package registry

import (
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	helmChartAPIVersion = "helm.cattle.io/v1"
	helmChartKind       = "HelmChart"
	helmChartSource     = "edge-image-builder"
	helmBackoffLimit    = 20
)

type HelmCRD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string            `yaml:"name"`
		Namespace   string            `yaml:"namespace,omitempty"`
		Annotations map[string]string `yaml:"annotations"`
	} `yaml:"metadata"`
	Spec struct {
		Version         string `yaml:"version"`
		ValuesContent   string `yaml:"valuesContent,omitempty"`
		ChartContent    string `yaml:"chartContent"`
		TargetNamespace string `yaml:"targetNamespace,omitempty"`
		CreateNamespace bool   `yaml:"createNamespace,omitempty"`
		BackOffLimit    int    `yaml:"backOffLimit"`
	} `yaml:"spec"`
}

func NewHelmCRD(chart *image.HelmChart, chartContent, valuesContent, repositoryURL string) *HelmCRD {
	// Some OCI registries (incl. oci://registry.suse.com/edge) use a `-chart` suffix
	// in the names of the charts which may conflict with .Release.Name references.
	name := strings.TrimSuffix(chart.Name, "-chart")
	if chart.ReleaseName != "" {
		name = chart.ReleaseName
	}

	return &HelmCRD{
		APIVersion: helmChartAPIVersion,
		Kind:       helmChartKind,
		Metadata: struct {
			Name        string            `yaml:"name"`
			Namespace   string            `yaml:"namespace,omitempty"`
			Annotations map[string]string `yaml:"annotations"`
		}{
			Name:      name,
			Namespace: chart.InstallationNamespace,
			Annotations: map[string]string{
				"edge.suse.com/source":         helmChartSource,
				"edge.suse.com/repository-url": repositoryURL,
			},
		},
		Spec: struct {
			Version         string `yaml:"version"`
			ValuesContent   string `yaml:"valuesContent,omitempty"`
			ChartContent    string `yaml:"chartContent"`
			TargetNamespace string `yaml:"targetNamespace,omitempty"`
			CreateNamespace bool   `yaml:"createNamespace,omitempty"`
			BackOffLimit    int    `yaml:"backOffLimit"`
		}{
			Version:         chart.Version,
			ValuesContent:   valuesContent,
			ChartContent:    chartContent,
			TargetNamespace: chart.TargetNamespace,
			CreateNamespace: chart.CreateNamespace,
			BackOffLimit:    helmBackoffLimit,
		},
	}
}
