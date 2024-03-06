package registry

import (
	"fmt"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const HelmChartAPIVersion = "helm.cattle.io/v1"
const HelmChartKind = "HelmChart"

type HelmCRD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace,omitempty"`
	} `yaml:"metadata"`
	Spec struct {
		Repo            string         `yaml:"repo,omitempty"`
		Chart           string         `yaml:"chart,omitempty"`
		Version         string         `yaml:"version"`
		Set             map[string]any `yaml:"set,omitempty"`
		ValuesContent   string         `yaml:"valuesContent,omitempty"`
		ChartContent    string         `yaml:"chartContent"`
		TargetNamespace string         `yaml:"targetNamespace,omitempty"`
		CreateNamespace bool           `yaml:"createNamespace,omitempty"`
	} `yaml:"spec"`
}

func (c *HelmCRD) parseSetArgs() []string {
	if len(c.Spec.Set) > 0 {
		return parseSetArgs("", c.Spec.Set)
	}

	return nil
}

func parseSetArgs(prefix string, m map[string]any) []string {
	var args []string

	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		switch value := v.(type) {
		case string, bool, int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32:
			args = append(args, fmt.Sprintf("%s=%v", fullKey, value))
		case []any:
			for i, item := range value {
				switch itemValue := item.(type) {
				case map[string]any:
					for innerKey, innerValue := range itemValue {
						formattedKey := fmt.Sprintf("%s[%d].%s", fullKey, i, innerKey)
						args = append(args, fmt.Sprintf("%s=%v", formattedKey, innerValue))
					}
				default:
					args = append(args, fmt.Sprintf("%s[%d]=%v", fullKey, i, itemValue))
				}
			}
		case map[string]any:
			args = append(args, parseSetArgs(fullKey, value)...)
		}
	}

	return args
}

func newHelmCRD(chart *image.HelmChart, chartContent, valuesContent string) HelmCRD {
	return HelmCRD{
		APIVersion: HelmChartAPIVersion,
		Kind:       HelmChartKind,
		Metadata: struct {
			Name      string `yaml:"name"`
			Namespace string `yaml:"namespace,omitempty"`
		}{
			Name:      chart.Name,
			Namespace: chart.InstallationNamespace,
		},
		Spec: struct {
			Repo            string         `yaml:"repo,omitempty"`
			Chart           string         `yaml:"chart,omitempty"`
			Version         string         `yaml:"version"`
			Set             map[string]any `yaml:"set,omitempty"`
			ValuesContent   string         `yaml:"valuesContent,omitempty"`
			ChartContent    string         `yaml:"chartContent"`
			TargetNamespace string         `yaml:"targetNamespace,omitempty"`
			CreateNamespace bool           `yaml:"createNamespace,omitempty"`
		}{
			Version:         chart.Version,
			ValuesContent:   valuesContent,
			ChartContent:    chartContent,
			TargetNamespace: chart.TargetNamespace,
			CreateNamespace: chart.CreateNamespace,
		},
	}
}
