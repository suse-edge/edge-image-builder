package helm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelmRepositoryName(t *testing.T) {
	tests := []struct {
		name           string
		repoURL        string
		chart          string
		expectedOutput string
	}{
		{
			name:           "OCI",
			repoURL:        "oci://registry-1.docker.io/bitnamicharts/apache",
			chart:          "apache",
			expectedOutput: "oci://registry-1.docker.io/bitnamicharts/apache",
		},
		{
			name:           "HTTP",
			repoURL:        "https://suse-edge.github.io/charts",
			chart:          "metallb",
			expectedOutput: "repo-metallb/metallb",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoName := repositoryName(test.repoURL, test.chart)
			assert.Equal(t, test.expectedOutput, repoName)
		})
	}
}

func TestAddRepoCommand(t *testing.T) {
	var buf bytes.Buffer
	cmd := addRepoCommand("kubevirt", "https://suse-edge.github.io/charts", &buf)

	assert.Equal(t, []string{
		"helm",
		"repo",
		"add",
		"repo-kubevirt",
		"https://suse-edge.github.io/charts",
	}, cmd.Args)

	assert.Equal(t, &buf, cmd.Stdout)
	assert.Equal(t, &buf, cmd.Stderr)
}

func TestPullCommand(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		chart        string
		version      string
		destDir      string
		expectedArgs []string
	}{
		{
			name:    "OCI repository",
			repo:    "oci://registry-1.docker.io/bitnamicharts/apache",
			version: "10.5.2",
			destDir: "charts",
			expectedArgs: []string{
				"helm",
				"pull",
				"oci://registry-1.docker.io/bitnamicharts/apache",
				"--version",
				"10.5.2",
				"--destination",
				"charts",
			},
		},
		{
			name:    "HTTP repository",
			repo:    "https://suse-edge.github.io/charts",
			chart:   "kubevirt",
			version: "0.2.1",
			destDir: "charts",
			expectedArgs: []string{
				"helm",
				"pull",
				"repo-kubevirt/kubevirt",
				"--version",
				"0.2.1",
				"--destination",
				"charts",
			},
		},
		{
			name: "OCI repository without optional args",
			repo: "oci://registry-1.docker.io/bitnamicharts/apache",
			expectedArgs: []string{
				"helm",
				"pull",
				"oci://registry-1.docker.io/bitnamicharts/apache",
			},
		},
		{
			name:  "HTTP repository without optional args",
			repo:  "https://suse-edge.github.io/charts",
			chart: "kubevirt",
			expectedArgs: []string{
				"helm",
				"pull",
				"repo-kubevirt/kubevirt",
			},
		},
	}

	var buf bytes.Buffer

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := pullCommand(test.chart, test.repo, test.version, test.destDir, &buf)

			assert.Equal(t, test.expectedArgs, cmd.Args)
			assert.Equal(t, &buf, cmd.Stdout)
			assert.Equal(t, &buf, cmd.Stderr)
		})
	}
}

func TestTemplateCommand(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		chart        string
		version      string
		kubeVersion  string
		valuesPath   string
		expectedArgs []string
	}{
		{
			name:        "Template with all parameters",
			repo:        "https://suse-edge.github.io/charts",
			chart:       "kubevirt",
			version:     "0.2.1",
			kubeVersion: "v1.29.0+rke2r1",
			valuesPath:  "/kubevirt/values.yaml",
			expectedArgs: []string{
				"helm",
				"template",
				"--skip-crds",
				"kubevirt",
				"https://suse-edge.github.io/charts",
				"--version",
				"0.2.1",
				"-f",
				"/kubevirt/values.yaml",
				"--kube-version",
				"v1.29.0+rke2r1",
			},
		},
		{
			name:        "Template without optional parameters",
			repo:        "https://suse-edge.github.io/charts",
			chart:       "kubevirt",
			kubeVersion: "v1.29.0+rke2r1",
			expectedArgs: []string{
				"helm",
				"template",
				"--skip-crds",
				"kubevirt",
				"https://suse-edge.github.io/charts",
				"--kube-version",
				"v1.29.0+rke2r1",
			},
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := templateCommand(test.chart, test.repo, test.version, test.valuesPath, test.kubeVersion, &stdout, &stderr)

			assert.Equal(t, test.expectedArgs, cmd.Args)
			assert.Equal(t, &stdout, cmd.Stdout)
			assert.Equal(t, &stderr, cmd.Stderr)
		})
	}
}

func TestParseChartContents_InvalidPayload(t *testing.T) {
	contents := "---abc"

	resources, err := parseChartContents(contents)
	require.Error(t, err)

	assert.ErrorContains(t, err, "invalid resource")
	assert.Nil(t, resources)
}

func TestParseChartContents(t *testing.T) {
	contents := `
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: metallb
  namespace: metallb-system
spec:
  repo: https://suse-edge.github.io/charts
  chart: metallb
---
apiVersion: v1
kind: Namespace
metadata:
  name: metallb-system
`

	resources, err := parseChartContents(contents)
	require.NoError(t, err)

	require.Len(t, resources, 2)

	assert.Equal(t, map[string]any{
		"apiVersion": "helm.cattle.io/v1",
		"kind":       "HelmChart",
		"metadata": map[string]any{
			"name":      "metallb",
			"namespace": "metallb-system",
		},
		"spec": map[string]any{
			"repo":  "https://suse-edge.github.io/charts",
			"chart": "metallb",
		},
	}, resources[0])

	assert.Equal(t, map[string]any{
		"apiVersion": "v1",
		"kind":       "Namespace",
		"metadata": map[string]any{
			"name": "metallb-system",
		},
	}, resources[1])
}
