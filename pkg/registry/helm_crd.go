package registry

import (
	"fmt"
)

const helmChartKind = "HelmChart"

type helmCRD struct {
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Repo            string         `yaml:"repo"`
		Chart           string         `yaml:"chart"`
		Version         string         `yaml:"version"`
		Set             map[string]any `yaml:"set"`
		ValuesContent   string         `yaml:"valuesContent"`
		ChartContent    string         `yaml:"chartContent"`
		TargetNamespace string         `yaml:"targetNamespace"`
		CreateNamespace bool           `yaml:"createNamespace"`
	} `yaml:"spec"`
}

func (c *helmCRD) parseSetArgs() []string {
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
