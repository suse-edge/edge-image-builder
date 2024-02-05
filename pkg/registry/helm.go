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
	Spec struct {
		Repo          string         `yaml:"repo"`
		Chart         string         `yaml:"chart"`
		Version       string         `yaml:"version"`
		Set           map[string]any `yaml:"set"`
		ValuesContent string         `yaml:"valuesContent"`
	} `yaml:"spec"`
}

var registryServiceURL = "http://hauler-registry.default.svc.cluster.local"

func parseSetArgs(prefix string, m map[string]any) []string {
	var args []string

	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		switch value := v.(type) {
		case string, bool, int, int64, float64: // TODO: probably include all primitive types
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

func parseHelmCRDs(manifestsPath string) ([]*HelmCRD, error) {
	crdFile, err := os.ReadFile(manifestsPath)
	if err != nil {
		return nil, fmt.Errorf("reading helm manifest: %w", err)
	}

	var crds []*HelmCRD

	decoder := yaml.NewDecoder(bytes.NewReader(crdFile))
	for {
		var manifest map[string]any

		if err = decoder.Decode(&manifest); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("unmarshaling manifest: %w", err)
		}

		kind, ok := manifest["kind"]
		if !ok {
			return nil, fmt.Errorf("missing 'kind' field in YAML document")
		}

		if kind != "HelmChart" {
			continue
		}

		yamlBytes, err := yaml.Marshal(manifest)
		if err != nil {
			return nil, fmt.Errorf("marshaling manifest: %w", err)
		}

		var crd HelmCRD
		if err = yaml.Unmarshal(yamlBytes, &crd); err != nil {
			return nil, fmt.Errorf("unmarshaling helm CRD: %w", err)
		}

		crds = append(crds, &crd)
	}

	if len(crds) == 0 {
		return nil, fmt.Errorf("no HelmChart found in the provided file")
	}

	return crds, nil
}

func GenerateHelmCommands(localHelmSrcDir string) ([]string, []string, error) {
	if localHelmSrcDir == "" {
		return nil, nil, nil
	}

	helmManifestPaths, err := getLocalManifestPaths(localHelmSrcDir)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting helm manifest paths: %w", err)
	}

	var helmCommands []string
	var helmChartPaths []string

	for _, manifest := range helmManifestPaths {
		helmCRDs, err := parseHelmCRDs(manifest)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing helm manifest: %w", err)
		}

		for _, crd := range helmCRDs {
			var valuesPath string

			if crd.Spec.ValuesContent != "" {
				valuesPath = fmt.Sprintf("values-%s.yaml", crd.Spec.Chart)

				if err = os.WriteFile(valuesPath, []byte(crd.Spec.ValuesContent), fileio.NonExecutablePerms); err != nil {
					return nil, nil, fmt.Errorf("writing helm values file: %w", err)
				}
			}

			tempRepo := fmt.Sprintf("repo-%s", crd.Spec.Chart)
			repository := helmRepositoryName(crd.Spec.Repo, tempRepo, crd.Spec.Chart)

			addCommand := helmAddRepoCommand(crd.Spec.Repo, tempRepo)
			if addCommand != "" {
				helmCommands = append(helmCommands, addCommand)
			}

			templateCommand := helmTemplateCommand(crd, repository, valuesPath)
			pullCommand := helmPullCommand(crd.Spec.Repo, crd.Spec.Chart, crd.Spec.Version)
			helmCommands = append(helmCommands, pullCommand, templateCommand)
			helmChartPaths = append(helmChartPaths, fmt.Sprintf("%s-*.tgz", crd.Spec.Chart))
		}
	}

	return helmCommands, helmChartPaths, nil
}

func helmRepositoryName(repoURL, tempRepo, chart string) string {
	if strings.HasPrefix(repoURL, "http") {
		return fmt.Sprintf("%s/%s", tempRepo, chart)
	}

	return repoURL
}

func helmAddRepoCommand(repo, tempRepo string) string {
	if !strings.HasPrefix(repo, "http") {
		return ""
	}

	return fmt.Sprintf("helm repo add %s %s", tempRepo, repo)
}

func helmPullCommand(repository, chart, version string) string {
	repository = helmRepositoryName(repository, fmt.Sprintf("repo-%s", chart), chart)

	pullCommand := fmt.Sprintf("helm pull %s", repository)
	if version != "" {
		pullCommand = fmt.Sprintf("helm pull %s --version %s", repository, version)
	}

	return pullCommand
}

func helmTemplateCommand(crd *HelmCRD, repository string, valuesFilePath string) string {
	var cmdParts []string

	cmdParts = append(cmdParts, fmt.Sprintf("helm template --skip-crds %s %s", crd.Spec.Chart, repository))

	if crd.Spec.Version != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--version %s", crd.Spec.Version))
	}

	if len(crd.Spec.Set) > 0 {
		args := parseSetArgs("", crd.Spec.Set)
		cmdParts = append(cmdParts, fmt.Sprintf("--set %s", strings.Join(args, ",")))
	}

	if crd.Spec.ValuesContent != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-f %s", valuesFilePath))
	}

	return strings.Join(cmdParts, " ")
}

func updateHelmManifest(manifestsPath string, chartTars []string) ([]map[string]interface{}, error) {
	manifestFile, err := os.ReadFile(manifestsPath)
	if err != nil {
		return nil, fmt.Errorf("reading helm manifest: %w", err)
	}

	var manifests []map[string]interface{}
	decoder := yaml.NewDecoder(bytes.NewReader(manifestFile))
	for {
		var manifest map[string]interface{}

		if err = decoder.Decode(&manifest); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("unmarshaling manifest: %w", err)
		}

		kind, ok := manifest["kind"]
		if !ok || kind != "HelmChart" {
			continue
		}

		if spec, ok := manifest["spec"].(map[string]interface{}); ok {
			delete(spec, "repo")
			oldChart := spec["chart"]
			for _, chartTar := range chartTars {
				if strings.Contains(chartTar, oldChart.(string)) {
					spec["chart"] = fmt.Sprintf("%s/%s", registryServiceURL, chartTar)
				}
			}

		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

func UpdateAllManifests(localHelmSrcDir string, chartTars []string) ([][]map[string]interface{}, error) {
	if localHelmSrcDir == "" {
		return nil, nil
	}

	helmManifestPaths, err := getLocalManifestPaths(localHelmSrcDir)
	if err != nil {
		return nil, fmt.Errorf("error getting helm manifest paths: %w", err)
	}

	var allManifests [][]map[string]interface{}
	for _, manifest := range helmManifestPaths {
		updatedManifests, err := updateHelmManifest(manifest, chartTars)
		if err != nil {
			return nil, fmt.Errorf("updating helm manifest: %w", err)
		}

		allManifests = append(allManifests, updatedManifests)
	}

	return allManifests, nil
}
