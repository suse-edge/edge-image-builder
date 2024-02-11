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

type HelmCRD struct {
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
	HelmChartKind = "HelmChart"
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
			return nil, fmt.Errorf("missing 'kind' field in helm manifest")
		}

		if kind != HelmChartKind {
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

func GenerateHelmCommands(localHelmSrcDir string, destDir string) (helmCommands []string, helmChartPaths []string, err error) {
	if localHelmSrcDir == "" {
		return nil, nil, nil
	}

	helmManifestPaths, err := getManifestPaths(localHelmSrcDir)
	if err != nil {
		return nil, nil, fmt.Errorf("getting helm manifest paths: %w", err)
	}

	for _, manifest := range helmManifestPaths {
		helmCRDs, err := parseHelmCRDs(manifest)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing helm manifest in '%s': %w", manifest, err)
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
				tempRepo := fmt.Sprintf("repo-%s", crd.Spec.Chart)
				repository := helmRepositoryName(crd.Spec.Repo, tempRepo, crd.Spec.Chart)

				addCommand := helmAddRepoCommand(crd.Spec.Repo, tempRepo)
				if addCommand != "" {
					helmCommands = append(helmCommands, addCommand)
				}

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

func helmPullCommand(repository, chart, version string, destDir string) string {
	repository = helmRepositoryName(repository, fmt.Sprintf("repo-%s", chart), chart)

	pullCommand := fmt.Sprintf("helm pull %s", repository)
	if version != "" {
		pullCommand = fmt.Sprintf("helm pull %s --version %s", repository, version)
	}
	if destDir != "" {
		pullCommand = fmt.Sprintf("%s -d %s", pullCommand, destDir)
	}

	return pullCommand
}

func helmTemplateCommand(crd *HelmCRD, repository string, valuesFilePath string, chartName string) string {
	var cmdParts []string

	cmdParts = append(cmdParts, fmt.Sprintf("helm template --skip-crds %s %s", chartName, repository))

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
		if !ok || kind != HelmChartKind {
			manifests = append(manifests, manifest)
			continue
		}

		if spec, ok := manifest["spec"].(map[string]any); ok {
			if _, ok := spec["chartContent"].(string); ok {
				manifests = append(manifests, manifest)
				continue
			}
			oldChart := spec["chart"]
			delete(spec, "repo")
			delete(spec, "chart")
			for _, chartTar := range chartTarsPaths {
				if strings.Contains(chartTar, oldChart.(string)) {
					tarData, err := os.ReadFile(chartTar)
					if err != nil {
						return nil, fmt.Errorf("reading chart tar '%s': %w", chartTar, err)
					}
					base64Str := base64.StdEncoding.EncodeToString(tarData)
					spec["chartContent"] = base64Str
				}
			}
		}

		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

func UpdateAllManifests(localHelmSrcDir string, chartTarsPath []string) ([][]map[string]any, error) {
	if localHelmSrcDir == "" {
		return nil, nil
	}

	helmManifestPaths, err := getManifestPaths(localHelmSrcDir)
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
