package registry

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const helmChartAPIVersion = "helm.cattle.io/v1"
const helmChartKind = "HelmChart"

type HelmCRD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace,omitempty"`
	} `yaml:"metadata"`
	Spec struct {
		Version         string `yaml:"version"`
		ValuesContent   string `yaml:"valuesContent,omitempty"`
		ChartContent    string `yaml:"chartContent"`
		TargetNamespace string `yaml:"targetNamespace,omitempty"`
		CreateNamespace bool   `yaml:"createNamespace,omitempty"`
	} `yaml:"spec"`
}

func NewHelmCRD(chart *image.HelmChart, chartContent, valuesContent string) HelmCRD {
	return HelmCRD{
		APIVersion: helmChartAPIVersion,
		Kind:       helmChartKind,
		Metadata: struct {
			Name      string `yaml:"name"`
			Namespace string `yaml:"namespace,omitempty"`
		}{
			Name:      chart.Name,
			Namespace: chart.InstallationNamespace,
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
