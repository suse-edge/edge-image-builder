package registry

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	helmChartAPIVersion = "helm.cattle.io/v1"
	helmChartKind       = "HelmChart"
	helmChartSource     = "edge-image-builder"
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
	} `yaml:"spec"`
}

func NewHelmCRD(chart *image.HelmChart, chartContent, valuesContent, repositoryURL string) HelmCRD {
	return HelmCRD{
		APIVersion: helmChartAPIVersion,
		Kind:       helmChartKind,
		Metadata: struct {
			Name        string            `yaml:"name"`
			Namespace   string            `yaml:"namespace,omitempty"`
			Annotations map[string]string `yaml:"annotations"`
		}{
			Name:      chart.Name,
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
		}{
			Version:         chart.Version,
			ValuesContent:   valuesContent,
			ChartContent:    chartContent,
			TargetNamespace: chart.TargetNamespace,
			CreateNamespace: chart.CreateNamespace,
		},
	}
}
