package registry

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"gopkg.in/yaml.v3"
)

type helmCRD struct {
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Repo          string         `yaml:"repo"`
		Chart         string         `yaml:"chart"`
		Version       string         `yaml:"version"`
		Set           map[string]any `yaml:"set"`
		ValuesContent string         `yaml:"valuesContent"`
		ChartContent  string         `yaml:"chartContent"`
	} `yaml:"spec"`
}

const (
	helmChartKind = "HelmChart"
)

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

func parseHelmCRDs(manifestsPath string) ([]*helmCRD, error) {
	crdFile, err := os.ReadFile(manifestsPath)
	if err != nil {
		return nil, fmt.Errorf("reading helm manifest: %w", err)
	}

	var crds []*helmCRD

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
			return nil, fmt.Errorf("missing 'kind' field in helm manifest")
		}

		if kind != helmChartKind {
			continue
		}

		yamlBytes, err := yaml.Marshal(manifest)
		if err != nil {
			return nil, fmt.Errorf("marshaling manifest: %w", err)
		}

		var crd helmCRD
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

func GenerateHelmCommands(localHelmSrcDir string, destDir string) (helmCommands []string, helmChartPaths []string, err error) {
	if localHelmSrcDir == "" {
		return nil, nil, nil
	}

	if destDir == "" {
		return nil, nil, fmt.Errorf("destination directory must be specified")
	}

	helmManifestPaths, err := getLocalManifestPaths(localHelmSrcDir)
	if err != nil {
		return nil, nil, fmt.Errorf("getting helm manifest paths: %w", err)
	}

	for _, manifest := range helmManifestPaths {
		helmCRDs, err := parseHelmCRDs(manifest)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing manifest '%s': %w", manifest, err)
		}

		for _, crd := range helmCRDs {
			var valuesPath string

			if crd.Spec.ValuesContent != "" {
				valuesPath = filepath.Join(destDir, fmt.Sprintf("values-%s.yaml", crd.Spec.Chart))

				if err = os.WriteFile(valuesPath, []byte(crd.Spec.ValuesContent), fileio.NonExecutablePerms); err != nil {
					return nil, nil, fmt.Errorf("writing helm values file: %w", err)
				}
			}

			if crd.Spec.ChartContent == "" {
				addCommand := helmAddRepoCommand(crd.Spec.Repo, crd.Spec.Chart)
				if addCommand != "" {
					helmCommands = append(helmCommands, addCommand)
				}

				repository := helmRepositoryName(crd.Spec.Repo, crd.Spec.Chart)
				templateCommand := helmTemplateCommand(crd, repository, valuesPath, crd.Spec.Chart)
				pullCommand := helmPullCommand(crd.Spec.Repo, crd.Spec.Chart, crd.Spec.Version, destDir)
				helmCommands = append(helmCommands, pullCommand, templateCommand)
				helmChartPaths = append(helmChartPaths, fmt.Sprintf("%s-*.tgz", crd.Spec.Chart))
			} else {
				decodedTar, err := base64.StdEncoding.DecodeString(crd.Spec.ChartContent)
				if err != nil {
					return nil, nil, fmt.Errorf("decoding base64 chart content: %w", err)
				}
				chartTar := filepath.Join(destDir, fmt.Sprintf("%s.tgz", crd.Metadata.Name))
				err = os.WriteFile(chartTar, decodedTar, fileio.NonExecutablePerms)
				if err != nil {
					return nil, nil, fmt.Errorf("writing decoded chart to file: %w", err)
				}

				templateCommand := helmTemplateCommand(crd, chartTar, valuesPath, crd.Metadata.Name)
				helmCommands = append(helmCommands, templateCommand)
				helmChartPaths = append(helmChartPaths, chartTar)
			}
		}
	}

	return helmCommands, helmChartPaths, nil
}

func helmTempRepo(chart string) string {
	return fmt.Sprintf("repo-%s", chart)
}

func helmRepositoryName(repoURL, chart string) string {
	if strings.HasPrefix(repoURL, "http") {
		return fmt.Sprintf("%s/%s", helmTempRepo(chart), chart)
	}

	return repoURL
}

func helmAddRepoCommand(repo, chart string) string {
	if !strings.HasPrefix(repo, "http") {
		return ""
	}

	var args []string
	args = append(args, "helm", "repo", "add", helmTempRepo(chart), repo)

	return strings.Join(args, " ")
}

func helmPullCommand(repository, chart, version string, destDir string) string {
	repository = helmRepositoryName(repository, chart)

	var args []string
	args = append(args, "helm", "pull", repository)

	if version != "" {
		args = append(args, "--version", version)
	}

	args = append(args, "-d", destDir)

	return strings.Join(args, " ")
}

func helmTemplateCommand(crd *helmCRD, repository string, valuesFilePath string, chartName string) string {
	var args []string
	args = append(args, "helm", "template", "--skip-crds", chartName, repository)

	if crd.Spec.Version != "" {
		args = append(args, "--version", crd.Spec.Version)
	}

	if len(crd.Spec.Set) > 0 {
		setArgs := parseSetArgs("", crd.Spec.Set)
		args = append(args, "--set", strings.Join(setArgs, ","))
	}

	if crd.Spec.ValuesContent != "" {
		args = append(args, "-f", valuesFilePath)
	}

	return strings.Join(args, " ")
}

func updateHelmManifest(manifestPath string, chartTarsPaths []string) ([]map[string]any, error) {
	manifestFile, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading helm manifest '%s': %w", manifestPath, err)
	}

	var manifests []map[string]any
	decoder := yaml.NewDecoder(bytes.NewReader(manifestFile))
	for {
		var manifest map[string]any

		if err = decoder.Decode(&manifest); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("unmarshaling manifest '%s': %w", manifestPath, err)
		}

		kind, ok := manifest["kind"]
		if !ok || kind != helmChartKind {
			manifests = append(manifests, manifest)
			continue
		}

		if spec, ok := manifest["spec"].(map[string]any); ok {
			if _, ok := spec["chartContent"].(string); ok {
				manifests = append(manifests, manifest)
				continue
			}
			chartName := spec["chart"]
			delete(spec, "repo")
			delete(spec, "chart")
			var noMatchingCharts = true
			for _, chartTar := range chartTarsPaths {
				if strings.Contains(chartTar, chartName.(string)) {
					noMatchingCharts = false
					tarData, err := os.ReadFile(chartTar)
					if err != nil {
						return nil, fmt.Errorf("reading chart tar '%s': %w", chartTar, err)
					}
					base64Str := base64.StdEncoding.EncodeToString(tarData)
					spec["chartContent"] = base64Str
				}
			}
			if noMatchingCharts {
				return nil, fmt.Errorf("no tarball path matching chart: '%s'", chartName)
			}
		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

func UpdateHelmManifests(localHelmSrcDir string, chartTarsPath []string) ([][]map[string]any, error) {
	if localHelmSrcDir == "" {
		return nil, nil
	}

	helmManifestPaths, err := getLocalManifestPaths(localHelmSrcDir)
	if err != nil {
		return nil, fmt.Errorf("getting helm manifest paths: %w", err)
	}

	var allManifests [][]map[string]any
	for _, manifest := range helmManifestPaths {
		updatedManifests, err := updateHelmManifest(manifest, chartTarsPath)
		if err != nil {
			return nil, fmt.Errorf("updating helm manifest: %w", err)
		}

		allManifests = append(allManifests, updatedManifests)
	}

	return allManifests, nil
}
