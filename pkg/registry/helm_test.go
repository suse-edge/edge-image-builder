package registry

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSetArgs(t *testing.T) {
	apacheManifestPath := filepath.Join("testdata", "helm", "oci.yaml")
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
			manifestPath: filepath.Join("testdata", "helm", "oci.yaml"),
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
			manifestPath: filepath.Join("testdata", "helm", "repo.yaml"),
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
			manifestPath: filepath.Join("testdata", "helm", "chart-content.yaml"),
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
			manifestPath:  filepath.Join("testdata", "helm", "invalid.yaml"),
			expectedError: "unmarshaling manifest: yaml:",
		},
		{
			name:          "No Kind",
			manifestPath:  filepath.Join("testdata", "helm", "no-kind.yaml"),
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
