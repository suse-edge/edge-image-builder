package registry

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParseSetArgs(t *testing.T) {
	apacheManifestPath := filepath.Join("testdata", "helm", "valid", "oci.yaml")
	apacheData, err := os.ReadFile(apacheManifestPath)
	require.NoError(t, err)
	var apacheManifest HelmCRD
	err = yaml.Unmarshal(apacheData, &apacheManifest) // Only gets the first doc
	require.NoError(t, err)

	complexManifestPath := filepath.Join("testdata", "helm", "complex-crd.yaml")
	complexData, err := os.ReadFile(complexManifestPath)
	require.NoError(t, err)
	var complexManifest HelmCRD
	err = yaml.Unmarshal(complexData, &complexManifest)
	require.NoError(t, err)

	tests := []struct {
		name         string
		manifestData map[string]any
		expectedSet  []string
	}{
		{
			name:         "Defined Set",
			manifestData: apacheManifest.Spec.Set,
			expectedSet:  []string{"ssl.enabled=true", "rbac.enabled=true", "servers[0].host=example", "servers[0].port=80", "servers[1].host=example2", "servers[1].port=22"},
		},
		{
			name:         "Undefined Set",
			manifestData: nil,
			expectedSet:  []string{},
		},
		{
			name:         "Complex Set",
			manifestData: complexManifest.Spec.Set,
			expectedSet: []string{
				"stringValue=exampleString",
				"boolValue=true",
				"intValue=42",
				"float32Value=3.14",
				"uintValue=7",
				"complexArray[0]=arrayString",
				"complexArray[1]=88",
				"complexArray[2]=false",
				"complexArray[3]=[map[name:nestedMap value:mapValue]]", // This type of nesting starts to fail
				"nestedMap.mapKey1=mapValue1",
				"nestedMap.mapKey2.nestedKey=nestedValue", // A map of maps with a key value still works
				"numericValues.int8Value=127",
				"numericValues.int16Value=32767",
				"numericValues.int32Value=2147483647",
				"numericValues.int64Value=9223372036854775807",
				"numericValues.float64Value=6.283185307179586",
				"numericValues.uint8Value=255",
				"numericValues.uint16Value=65535",
				"numericValues.uint32Value=4294967295",
				"servers[0].port=80", // Array of maps
				"servers[0].host=example.com",
				"servers[1].port=443",
				"servers[1].host=secure.example.com",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parseSetOutput := parseSetArgs("", test.manifestData)
			assert.ElementsMatch(t, test.expectedSet, parseSetOutput)
		})
	}
}

func TestParseHelmCRDs(t *testing.T) {
	tests := []struct {
		name          string
		manifestPath  string
		expectedCRD   HelmCRD
		expectedError string
	}{
		{
			name:         "OCI With Values",
			manifestPath: filepath.Join("testdata", "helm", "valid", "oci.yaml"),
			expectedCRD: HelmCRD{
				Metadata: struct {
					Name string `yaml:"name"`
				}{
					Name: "apache",
				},
				Spec: struct {
					Repo          string         `yaml:"repo"`
					Chart         string         `yaml:"chart"`
					Version       string         `yaml:"version"`
					Set           map[string]any `yaml:"set"`
					ValuesContent string         `yaml:"valuesContent"`
					ChartContent  string         `yaml:"chartContent"`
				}{
					Repo:    "oci://registry-1.docker.io/bitnamicharts/apache",
					Chart:   "apache",
					Version: "10.5.2",
					Set: map[string]interface{}{
						"rbac.enabled": "true",
						"ssl.enabled":  "true",
						"servers": []interface{}{
							map[string]interface{}{"host": "example", "port": 80},
							map[string]interface{}{"host": "example2", "port": 22},
						},
					},
					ValuesContent: `service:
  type: ClusterIP
ingress:
  enabled: true
  hostname: www.example.com
metrics:
  enabled: true`,
				},
			},
		},
		{
			name:         "HTTP Repo",
			manifestPath: filepath.Join("testdata", "helm", "valid", "repo.yaml"),
			expectedCRD: HelmCRD{
				Metadata: struct {
					Name string `yaml:"name"`
				}{
					Name: "metallb",
				},
				Spec: struct {
					Repo          string         `yaml:"repo"`
					Chart         string         `yaml:"chart"`
					Version       string         `yaml:"version"`
					Set           map[string]any `yaml:"set"`
					ValuesContent string         `yaml:"valuesContent"`
					ChartContent  string         `yaml:"chartContent"`
				}{
					Repo:    "https://suse-edge.github.io/charts",
					Chart:   "metallb",
					Version: "",
					Set:     nil,
				},
			},
		},
		{
			name:         "Chart Content",
			manifestPath: filepath.Join("testdata", "helm", "valid", "chart-content.yaml"),
			expectedCRD: HelmCRD{
				Metadata: struct {
					Name string `yaml:"name"`
				}{
					Name: "apache2",
				},
				Spec: struct {
					Repo          string         `yaml:"repo"`
					Chart         string         `yaml:"chart"`
					Version       string         `yaml:"version"`
					Set           map[string]any `yaml:"set"`
					ValuesContent string         `yaml:"valuesContent"`
					ChartContent  string         `yaml:"chartContent"`
				}{
					ChartContent: "H4sIFAAAAAAA",
				},
			},
		},
		{
			name:          "None Existent Path",
			manifestPath:  filepath.Join("testdata", "helm", "dne-yaml"),
			expectedError: "reading helm manifest: open testdata/helm/dne-yaml: no such file or directory",
		},
		{
			name:          "Empty File",
			manifestPath:  filepath.Join("testdata", "empty-crd.yaml"),
			expectedError: "no HelmChart found in the provided file",
		},
		{
			name:          "Invalid File",
			manifestPath:  filepath.Join("testdata", "helm", "invalid", "invalid.yaml"),
			expectedError: "unmarshaling manifest: yaml:",
		},
		{
			name:          "No Kind",
			manifestPath:  filepath.Join("testdata", "helm", "invalid", "no-kind.yaml"),
			expectedError: "missing 'kind' field in helm manifest",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parseHelmCRDsOutput, err := parseHelmCRDs(test.manifestPath)
			if test.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, test.expectedCRD, *parseHelmCRDsOutput[0])
			} else {
				require.ErrorContains(t, err, test.expectedError)
			}
		})
	}
}

func TestUpdateHelmManifest(t *testing.T) {
	ociManifestPath := filepath.Join("testdata", "helm", "valid", "oci.yaml")
	chartContentPath := filepath.Join("testdata", "helm", "valid", "chart-content.yaml")
	repoManifestPath := filepath.Join("testdata", "helm", "valid", "repo.yaml")
	invalidManifestPath := filepath.Join("testdata", "helm", "invalid", "invalid.yaml")

	chartTarPaths := []string{
		filepath.Join("testdata", "helm", "apache-10.5.2.tgz"),
		filepath.Join("testdata", "helm", "metallb.tgz"),
	}

	tests := []struct {
		name                string
		manifestPath        string
		nonHelmExpectedKind string
		chartTarPaths       []string
		tarTest             bool
		expectedError       string
	}{
		{
			name:                "OCI Manifest",
			manifestPath:        ociManifestPath,
			chartTarPaths:       chartTarPaths,
			nonHelmExpectedKind: "Namespace",
		},
		{
			name:          "Chart Content Manifest",
			manifestPath:  chartContentPath,
			chartTarPaths: chartTarPaths,
		},
		{
			name:          "Helm Repo Manifest",
			manifestPath:  repoManifestPath,
			chartTarPaths: chartTarPaths,
		},
		{
			name:          "Nonexistent Path",
			manifestPath:  "dne",
			chartTarPaths: chartTarPaths,
			expectedError: "reading helm manifest 'dne': open dne: no such file or directory",
		},
		{
			name:          "Invalid Manifest",
			manifestPath:  invalidManifestPath,
			chartTarPaths: chartTarPaths,
			expectedError: "unmarshaling manifest 'testdata/helm/invalid/invalid.yaml': yaml:",
		},
		{
			// No error because the name of the chart must be present in the name of the chart tar
			name:          "Invalid Chart Tar Paths",
			manifestPath:  repoManifestPath,
			chartTarPaths: []string{"non-existent-tar"},
			tarTest:       true,
		},
		{
			name:          "Invalid Chart Tar",
			manifestPath:  repoManifestPath,
			chartTarPaths: []string{"metallb-invalid-path"},
			tarTest:       true,
			expectedError: "reading chart tar 'metallb-invalid-path': open metallb-invalid-path: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest, err := updateHelmManifest(test.manifestPath, test.chartTarPaths)
			if test.expectedError == "" && !test.tarTest {
				require.NoError(t, err)
				for _, doc := range manifest {
					if spec, ok := doc["spec"].(map[string]any); ok {
						assert.NotEmpty(t, spec)
						chartContent, ok := spec["chartContent"].(string)
						assert.Equal(t, true, ok)
						assert.NotEmpty(t, chartContent)
					} else {
						kind, ok := doc["kind"].(string)
						assert.Equal(t, true, ok)
						assert.Equal(t, test.nonHelmExpectedKind, kind)
					}
				}
			} else {
				if test.expectedError != "" {
					require.ErrorContains(t, err, test.expectedError)
				} else {
					require.NoError(t, err)
				}
			}

		})
	}
}

func TestHelmRepositoryName(t *testing.T) {
	tests := []struct {
		name           string
		repoURL        string
		tempRepo       string
		chart          string
		expectedOutput string
	}{
		{
			name:           "OCI",
			repoURL:        "oci://registry-1.docker.io/bitnamicharts/apache",
			tempRepo:       "",
			chart:          "apache",
			expectedOutput: "oci://registry-1.docker.io/bitnamicharts/apache",
		},
		{
			name:           "HTTP Repo",
			repoURL:        "https://suse-edge.github.io/charts",
			tempRepo:       "tempRepo",
			chart:          "metallb",
			expectedOutput: "tempRepo/metallb",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoName := helmRepositoryName(test.repoURL, test.tempRepo, test.chart)
			assert.Equal(t, test.expectedOutput, repoName)
		})
	}
}

func TestHelmAddRepoCommand(t *testing.T) {
	tests := []struct {
		name           string
		repoURL        string
		tempRepo       string
		expectedOutput string
	}{
		{
			name:           "OCI",
			repoURL:        "oci://registry-1.docker.io/bitnamicharts/apache",
			tempRepo:       "tempRepo",
			expectedOutput: "",
		},
		{
			name:           "HTTP Repo",
			repoURL:        "https://suse-edge.github.io/charts",
			tempRepo:       "tempRepo",
			expectedOutput: "helm repo add tempRepo https://suse-edge.github.io/charts",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addRepoCommand := helmAddRepoCommand(test.repoURL, test.tempRepo)
			assert.Equal(t, test.expectedOutput, addRepoCommand)
		})
	}
}

func TestHelmPullCommand(t *testing.T) {
	tests := []struct {
		name           string
		repo           string
		chart          string
		version        string
		destDir        string
		expectedOutput string
	}{
		{
			name:           "OCI",
			repo:           "oci://registry-1.docker.io/bitnamicharts/apache",
			chart:          "",
			destDir:        "helm",
			expectedOutput: "helm pull oci://registry-1.docker.io/bitnamicharts/apache -d helm",
		},
		{
			name:           "HTTP Repo",
			repo:           "https://suse-edge.github.io/charts",
			chart:          "sample-chart",
			destDir:        "",
			expectedOutput: "helm pull repo-sample-chart/sample-chart",
		},
		{
			name:           "OCI",
			repo:           "oci://registry-1.docker.io/bitnamicharts/apache",
			chart:          "",
			version:        "10.5.2",
			expectedOutput: "helm pull oci://registry-1.docker.io/bitnamicharts/apache --version 10.5.2",
		},
		{
			name:           "HTTP Repo",
			repo:           "https://suse-edge.github.io/charts",
			chart:          "sample-chart",
			version:        "1.0.0",
			destDir:        "helmDir/helmcharts",
			expectedOutput: "helm pull repo-sample-chart/sample-chart --version 1.0.0 -d helmDir/helmcharts",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoName := helmPullCommand(test.repo, test.chart, test.version, test.destDir)
			assert.Equal(t, test.expectedOutput, repoName)
		})
	}
}

func TestHelmTemplateCommand(t *testing.T) {
	tests := []struct {
		name           string
		crd            *HelmCRD
		repo           string
		valuesFilePath string
		chart          string
		expectedOutput string
	}{
		{
			name: "OCI",
			repo: "oci://registry-1.docker.io/bitnamicharts/apache",
			crd: &HelmCRD{
				Metadata: struct {
					Name string `yaml:"name"`
				}{
					Name: "apache",
				},
				Spec: struct {
					Repo          string         `yaml:"repo"`
					Chart         string         `yaml:"chart"`
					Version       string         `yaml:"version"`
					Set           map[string]any `yaml:"set"`
					ValuesContent string         `yaml:"valuesContent"`
					ChartContent  string         `yaml:"chartContent"`
				}{
					Repo:    "oci://registry-1.docker.io/bitnamicharts/apache",
					Chart:   "apache",
					Version: "10.5.2",
					Set: map[string]interface{}{
						"rbac.enabled": "true",
						"ssl.enabled":  "true",
						"servers": []interface{}{
							map[string]interface{}{"host": "example", "port": 80},
							map[string]interface{}{"host": "example2", "port": 22},
						},
					},
					ValuesContent: `service:
 type: ClusterIP
ingress:
 enabled: true
 hostname: www.example.com
metrics:
 enabled: true`,
				},
			},
			valuesFilePath: "values.yaml",
			chart:          "apache",
			expectedOutput: "helm template --skip-crds apache oci://registry-1.docker.io/bitnamicharts/apache --version 10.5.2 --set rbac.enabled=true,ssl.enabled=true,servers[0].host=example,servers[0].port=80,servers[1].host=example2,servers[1].port=22 -f values.yaml",
		},
		{
			name: "HTTP Repo",
			repo: "repo-metallb/metallb",
			crd: &HelmCRD{
				Metadata: struct {
					Name string `yaml:"name"`
				}{
					Name: "metallb",
				},
				Spec: struct {
					Repo          string         `yaml:"repo"`
					Chart         string         `yaml:"chart"`
					Version       string         `yaml:"version"`
					Set           map[string]any `yaml:"set"`
					ValuesContent string         `yaml:"valuesContent"`
					ChartContent  string         `yaml:"chartContent"`
				}{
					Repo:  "https://suse-edge.github.io/charts",
					Chart: "metallb",
					Set: map[string]interface{}{
						"rbac.enabled": "true",
						"ssl.enabled":  "true",
					},
				},
			},
			chart:          "metallb",
			expectedOutput: "helm template --skip-crds metallb repo-metallb/metallb --set rbac.enabled=true,ssl.enabled=true",
		},
		{
			name: "Chart Tar",
			repo: "apache-10.5.2.tgz",
			crd: &HelmCRD{
				Metadata: struct {
					Name string `yaml:"name"`
				}{
					Name: "apache2",
				},
				Spec: struct {
					Repo          string         `yaml:"repo"`
					Chart         string         `yaml:"chart"`
					Version       string         `yaml:"version"`
					Set           map[string]any `yaml:"set"`
					ValuesContent string         `yaml:"valuesContent"`
					ChartContent  string         `yaml:"chartContent"`
				}{
					ChartContent: "H4sIFAAAAAAA",
					Version:      "10.5.2",
				},
			},
			chart:          "apache2",
			expectedOutput: "helm template --skip-crds apache2 apache-10.5.2.tgz --version 10.5.2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			templateCommand := helmTemplateCommand(test.crd, test.repo, test.valuesFilePath, test.chart)

			actualFields := strings.Fields(templateCommand)
			expectedFields := strings.Fields(test.expectedOutput)

			if !slices.Contains(actualFields, "--set") {
				assert.ElementsMatch(t, actualFields, expectedFields)
			} else {
				setIndex := slices.Index(actualFields, "--set")
				setValuesIndex := setIndex + 1

				actualSetFields := strings.Split(actualFields[setValuesIndex], ",")
				expectedSetFields := strings.Split(expectedFields[setValuesIndex], ",")

				assert.ElementsMatch(t, actualSetFields, expectedSetFields)
			}
		})
	}
}

func TestUpdateAllHelmManifest(t *testing.T) {
	localHelmSrcDirValid := filepath.Join("testdata", "helm", "valid")
	localHelmSrcDirInvalid := filepath.Join("testdata", "helm", "invalid")

	chartTarPaths := []string{
		filepath.Join("testdata", "helm", "apache-10.5.2.tgz"),
		filepath.Join("testdata", "helm", "metallb.tgz"),
	}

	tests := []struct {
		name           string
		helmSrcDir     string
		chartTarPaths  []string
		expectedOutput string
		expectedError  string
	}{
		{
			name:          "Helm Dir With Valid Manifests",
			helmSrcDir:    localHelmSrcDirValid,
			chartTarPaths: chartTarPaths,
			expectedError: "",
		},
		{
			name:          "Helm Dir With Invalid Manifests",
			helmSrcDir:    localHelmSrcDirInvalid,
			chartTarPaths: chartTarPaths,
			expectedError: "updating helm manifest: unmarshaling manifest 'testdata/helm/invalid/invalid.yaml': yaml:",
		},
		{
			name:       "Invalid Chart Tar",
			helmSrcDir: localHelmSrcDirValid,
			chartTarPaths: []string{
				"apache", // Chart tar name must include the name of one of the charts
			},
			expectedError: "updating helm manifest: reading chart tar 'apache': open apache: no such file or directory",
		},
		{
			name:       "No Helm Dir",
			helmSrcDir: "", // Returns nil
		},
		{
			name:          "Invalid Helm Dir",
			helmSrcDir:    "invalid",
			expectedError: "getting helm manifest paths: reading manifest source dir 'invalid': open invalid: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			allManifests, err := UpdateAllManifests(test.helmSrcDir, test.chartTarPaths)
			if test.expectedError == "" {
				require.NoError(t, err)
				for _, manifest := range allManifests {
					for _, doc := range manifest {
						spec, ok := doc["spec"].(map[string]any)
						if ok {
							assert.NotEmpty(t, spec)

							chartContent, ok := spec["chartContent"].(string)
							assert.True(t, ok)
							assert.NotEmpty(t, chartContent)

							chartName, ok := spec["chart"].(string)
							assert.False(t, ok)
							assert.Empty(t, chartName)

							repo, ok := spec["repo"].(string)
							assert.False(t, ok)
							assert.Empty(t, repo)
						}
					}
				}
			} else {
				require.ErrorContains(t, err, test.expectedError)
			}
		})
	}
}

func TestGenerateHelmCommands(t *testing.T) {
	helmDir, err := os.MkdirTemp("", "helm")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(helmDir))
	}()

	localHelmSrcDirValid := filepath.Join("testdata", "helm", "valid")
	localHelmSrcDirInvalid := filepath.Join("testdata", "helm", "invalid")

	tests := []struct {
		name             string
		helmSrcDir       string
		expectedCommands []string
		helmChartPaths   []string
		expectedError    string
	}{
		{
			name:       "Helm Dir With Valid Manifests",
			helmSrcDir: localHelmSrcDirValid,
			helmChartPaths: []string{
				filepath.Join(helmDir, "apache2.tgz"), // This one is created manually while the others are pulled
				"apache-*.tgz",
				"metallb-*.tgz",
			},
			expectedCommands: []string{
				fmt.Sprintf("helm template --skip-crds apache2 %s", filepath.Join(helmDir, "apache2.tgz")),
				fmt.Sprintf("helm pull oci://registry-1.docker.io/bitnamicharts/apache --version 10.5.2 -d %s", helmDir),
				fmt.Sprintf("helm template --skip-crds apache oci://registry-1.docker.io/bitnamicharts/apache --version 10.5.2 --set rbac.enabled=true,servers[0].host=example,servers[0].port=80,servers[1].host=example2,servers[1].port=22,ssl.enabled=true -f %s", filepath.Join(helmDir, "values-apache.yaml")),
				"helm template --skip-crds metallb repo-metallb/metallb",
				"helm repo add repo-metallb https://suse-edge.github.io/charts",
				fmt.Sprintf("helm pull repo-metallb/metallb -d %s", helmDir),
				"helm template --skip-crds metallb repo-metallb/metallb",
			},
		},
		{
			name:          "Helm Dir With Invalid Manifests",
			helmSrcDir:    localHelmSrcDirInvalid,
			expectedError: "parsing helm manifest in 'testdata/helm/invalid/invalid.yaml'",
		},
		{
			name:       "No Source Dir",
			helmSrcDir: "",
		},
		{
			name:          "Invalid Source Dir",
			helmSrcDir:    "invalid",
			expectedError: "getting helm manifest paths: reading manifest source dir 'invalid': open invalid: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			helmCommands, helmChartPaths, err := GenerateHelmCommands(test.helmSrcDir, helmDir)
			fmt.Println(err)
			if test.expectedError == "" {
				assert.ElementsMatch(t, helmChartPaths, test.helmChartPaths)
				require.NoError(t, err)
				for _, command := range helmCommands {
					if !strings.Contains(command, "--set") {
						assert.Contains(t, test.expectedCommands, command)
					} else {
						actualFields := strings.Fields(command)
						expectedFields := strings.Fields(test.expectedCommands[2])

						if !slices.Contains(actualFields, "--set") {
							assert.ElementsMatch(t, actualFields, expectedFields)
						} else {
							setIndex := slices.Index(actualFields, "--set")
							setValuesIndex := setIndex + 1

							actualSetFields := strings.Split(actualFields[setValuesIndex], ",")
							expectedSetFields := strings.Split(expectedFields[setValuesIndex], ",")

							assert.ElementsMatch(t, actualSetFields, expectedSetFields)
						}
					}
				}
			} else {
				require.ErrorContains(t, err, test.expectedError)
			}
		})
	}
	assert.FileExists(t, tests[0].helmChartPaths[0])
	assert.FileExists(t, filepath.Join(helmDir, "values-apache.yaml"))
}
