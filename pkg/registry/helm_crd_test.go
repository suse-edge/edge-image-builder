package registry

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseSetArgs_Empty(t *testing.T) {
	var crd HelmCRD

	assert.Empty(t, crd.parseSetArgs())
}

func TestParseSetArgs_Simple(t *testing.T) {
	data := `spec:
  repo: oci://registry-1.docker.io/bitnamicharts/apache
  chart: apache
  targetNamespace: web
  version: 10.5.2
  set:
    rbac.enabled: "true"
    ssl.enabled: "true"
    servers:
      - port: 80
        host: example
      - port: 22
        host: example2`

	var crd HelmCRD
	require.NoError(t, yaml.Unmarshal([]byte(data), &crd))

	expectedArgs := []string{
		"ssl.enabled=true",
		"rbac.enabled=true",
		"servers[0].host=example",
		"servers[0].port=80",
		"servers[1].host=example2",
		"servers[1].port=22",
	}

	assert.ElementsMatch(t, expectedArgs, crd.parseSetArgs())
}

func TestParseSetArgs_Complex(t *testing.T) {
	data := `spec:
  repo: oci://my-helm-chart-repository
  chart: my-chart
  targetNamespace: example-namespace
  version: "1.0.0"
  set:
    stringValue: "exampleString"
    boolValue: true
    intValue: 42
    float32Value: 3.14
    uintValue: 7
    complexArray:
      - "arrayString"
      - 88
      - false
      - - name: "nestedMap"
          value: "mapValue"
    nestedMap:
      mapKey1: "mapValue1"
      mapKey2:
        nestedKey: "nestedValue"
    numericValues:
      int8Value: 127
      int16Value: 32767
      int32Value: 2147483647
      int64Value: 9223372036854775807
      float64Value: 6.283185307179586
      uint8Value: 255
      uint16Value: 65535
      uint32Value: 4294967295
    servers:
      - port: 80
        host: "example.com"
      - port: 443
        host: "secure.example.com"`

	var crd HelmCRD
	require.NoError(t, yaml.Unmarshal([]byte(data), &crd))

	expectedArgs := []string{
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
	}

	assert.ElementsMatch(t, expectedArgs, crd.parseSetArgs())
}

func TestNewHelmCRD(t *testing.T) {
	chart := &image.HelmChart{
		Name:                  "apache",
		Repo:                  "oci://registry-1.docker.io/bitnamicharts/apache",
		TargetNamespace:       "web",
		CreateNamespace:       true,
		InstallationNamespace: "kube-system",
		Version:               "10.7.0",
		ValuesFile:            "apache-values.yaml",
	}
	chartContent := "Hxxxx"
	valuesContent := `
values: content`

	expectedCRD := HelmCRD{
		APIVersion: HelmChartAPIVersion,
		Kind:       HelmChartKind,
		Metadata: struct {
			Name      string `yaml:"name"`
			Namespace string `yaml:"namespace,omitempty"`
		}{
			Name:      "apache",
			Namespace: "kube-system",
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
			Version: "10.7.0",
			ValuesContent: `
values: content`,
			ChartContent:    "Hxxxx",
			TargetNamespace: "web",
			CreateNamespace: true,
		},
	}

	assert.Equal(t, expectedCRD, newHelmCRD(chart, chartContent, valuesContent))
}

func TestNewHelmCRDNoValues(t *testing.T) {
	chart := &image.HelmChart{
		Name:                  "apache",
		Repo:                  "oci://registry-1.docker.io/bitnamicharts/apache",
		TargetNamespace:       "web",
		CreateNamespace:       true,
		InstallationNamespace: "kube-system",
		Version:               "10.7.0",
		ValuesFile:            "apache-values.yaml",
	}
	chartContent := "Hxxxx"

	expectedCRD := HelmCRD{
		APIVersion: HelmChartAPIVersion,
		Kind:       HelmChartKind,
		Metadata: struct {
			Name      string `yaml:"name"`
			Namespace string `yaml:"namespace,omitempty"`
		}{
			Name:      "apache",
			Namespace: "kube-system",
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
			Version:         "10.7.0",
			ChartContent:    "Hxxxx",
			TargetNamespace: "web",
			CreateNamespace: true,
		},
	}

	assert.Equal(t, expectedCRD, newHelmCRD(chart, chartContent, ""))
}
