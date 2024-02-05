package registry

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"gopkg.in/yaml.v3"
)

type HelmCRD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Repo            string         `yaml:"repo"`
		Chart           string         `yaml:"chart"`
		TargetNamespace string         `yaml:"targetNamespace"`
		Version         string         `yaml:"version"`
		Set             map[string]any `yaml:"set"`
		ValuesContent   string         `yaml:"valuesContent"`
	} `yaml:"spec"`
}

func formatMap(prefix string, m map[string]any) string {
	var parts []string
	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		switch value := v.(type) {
		case string, bool, int, float64:
			parts = append(parts, fmt.Sprintf("%s=%v", fullKey, value))
		case []any:
			for i, item := range value {
				switch itemValue := item.(type) {
				case map[string]any:
					for innerKey, innerValue := range itemValue {
						formattedKey := fmt.Sprintf("%s[%d].%s", fullKey, i, innerKey)
						parts = append(parts, fmt.Sprintf("%s=%v", formattedKey, innerValue))
					}
				default:
					parts = append(parts, fmt.Sprintf("%s[%d]=%v", fullKey, i, itemValue))
				}
			}
		case map[string]any:
			parts = append(parts, formatMap(fullKey, value))
		}
	}

	return strings.Join(parts, ",")
}

func writeHelmValuesFile(crd *HelmCRD, valuesPath string) error {
	if valuesPath == "" {
		return nil
	}

	err := os.WriteFile(valuesPath, []byte(crd.Spec.ValuesContent), fileio.NonExecutablePerms)
	if err != nil {
		return fmt.Errorf("failed to write helm values file: %w", err)
	}

	return nil
}

func readAndParseHelmCRD(helmPath string) (HelmCRD, error) {
	crdFile, err := os.ReadFile(helmPath)
	if err != nil {
		return HelmCRD{}, fmt.Errorf("reading helm crd: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(crdFile))
	for {
		var genericMap map[string]any
		err := decoder.Decode(&genericMap)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return HelmCRD{}, fmt.Errorf("unmarshaling helm CRD to generic map: %w", err)
		}

		kind, ok := genericMap["kind"]
		if !ok {
			return HelmCRD{}, fmt.Errorf("missing 'kind' field in YAML document")
		}

		if kind == "HelmChart" {
			yamlBytes, err := yaml.Marshal(genericMap)
			if err != nil {
				return HelmCRD{}, fmt.Errorf("re-marshaling generic map to YAML: %w", err)
			}

			var crd HelmCRD
			err = yaml.Unmarshal(yamlBytes, &crd)
			if err != nil {
				return HelmCRD{}, fmt.Errorf("unmarshaling helm CRD to helm struct: %w", err)
			}
			return crd, nil
		}
	}

	return HelmCRD{}, fmt.Errorf("no HelmChart found in the provided file")
}

func GenerateHelmCommandsAndWriteHelmValues(localHelmSrcDir string) ([]string, []string, error) {
	var helmCommands []string
	var helmManifestPaths []string
	var err error
	var helmChartPaths []string

	if localHelmSrcDir != "" {
		helmManifestPaths, err = getLocalManifestPaths(localHelmSrcDir)
		if err != nil {
			return nil, nil, fmt.Errorf("error getting helm manifest paths: %w", err)
		}
	}

	for index, helmManifest := range helmManifestPaths {
		var valuesPath string
		helmCRD, err := readAndParseHelmCRD(helmManifest)
		if err != nil {
			return nil, nil, fmt.Errorf("reading and parsing helm crd: %w", err)
		}

		if helmCRD.Spec.ValuesContent != "" {
			valuesPath = fmt.Sprintf("values-%d.yaml", index)
		}

		var tempRepoWithChart string

		if strings.HasPrefix(helmCRD.Spec.Repo, "http") {
			tempRepo := fmt.Sprintf("repo-%d", index)
			repoCommand := fmt.Sprintf("helm repo add %s %s", tempRepo, helmCRD.Spec.Repo)
			tempRepoWithChart = fmt.Sprintf("repo-%d/%s", index, helmCRD.Spec.Chart)

			helmCommands = append(helmCommands, repoCommand)
			var pullCommand string
			if helmCRD.Spec.Version != "" {
				pullCommand = fmt.Sprintf("helm pull repo-%d/%s --version %s", index, helmCRD.Spec.Chart, helmCRD.Spec.Version)
			} else {
				pullCommand = fmt.Sprintf("helm pull repo-%d/%s", index, helmCRD.Spec.Chart)
			}
			helmCommands = append(helmCommands, pullCommand)
		} else {
			var pullCommand string
			if helmCRD.Spec.Version != "" {
				pullCommand = fmt.Sprintf("helm pull %s --version %s", helmCRD.Spec.Repo, helmCRD.Spec.Version)
			} else {
				pullCommand = fmt.Sprintf("helm pull %s", helmCRD.Spec.Repo)
			}
			helmCommands = append(helmCommands, pullCommand)
		}

		helmChartPaths = append(helmChartPaths, fmt.Sprintf("%s-*.tgz", helmCRD.Spec.Chart))
		helmCommand := buildHelmCommand(&helmCRD, valuesPath, tempRepoWithChart)
		err = writeHelmValuesFile(&helmCRD, valuesPath)
		if err != nil {
			return nil, nil, fmt.Errorf("writing helm values manifest: %w", err)
		}

		helmCommands = append(helmCommands, helmCommand)
	}

	return helmCommands, helmChartPaths, nil
}

func buildHelmCommand(crd *HelmCRD, valuesFilePath string, tempRepoWithChart string) string {
	var cmdParts []string

	if tempRepoWithChart != "" {
		cmdParts = append(cmdParts, "helm template --skip-crds", fmt.Sprintf("%s %s", crd.Spec.Chart, tempRepoWithChart))
	} else {
		cmdParts = append(cmdParts, "helm template --skip-crds", fmt.Sprintf("%s %s", crd.Spec.Chart, crd.Spec.Repo))
	}

	if crd.Spec.Version != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--version %s", crd.Spec.Version))
	}

	set := formatMap("", crd.Spec.Set)
	if len(crd.Spec.Set) > 0 {
		cmdParts = append(cmdParts, fmt.Sprintf("--set %s", set))
	}

	if crd.Spec.ValuesContent != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-f %s", valuesFilePath))
	}

	return strings.Join(cmdParts, " ")
}
