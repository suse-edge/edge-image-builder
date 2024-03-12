package helm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestHelmRepositoryName(t *testing.T) {
	tests := []struct {
		name           string
		repoName       string
		repoURL        string
		chart          string
		expectedOutput string
	}{
		{
			name:           "OCI",
			repoName:       "apache-repo",
			repoURL:        "oci://registry-1.docker.io/bitnamicharts/apache",
			chart:          "apache",
			expectedOutput: "oci://registry-1.docker.io/bitnamicharts/apache",
		},
		{
			name:           "HTTP",
			repoName:       "suse-edge",
			repoURL:        "https://suse-edge.github.io/charts",
			chart:          "metallb",
			expectedOutput: "suse-edge/metallb",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoName := repositoryName(test.repoName, test.repoURL, test.chart)
			assert.Equal(t, test.expectedOutput, repoName)
		})
	}
}

func TestAddRepoCommand(t *testing.T) {
	tests := []struct {
		name         string
		repo         *image.HelmRepository
		expectedArgs []string
	}{
		{
			name: "Valid Repository",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
			},
			expectedArgs: []string{
				"helm",
				"repo",
				"add",
				"suse-edge",
				"https://suse-edge.github.io/charts",
			},
		},
		{
			name: "Valid Repository With Auth",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
			},
			expectedArgs: []string{
				"helm",
				"repo",
				"add",
				"suse-edge",
				"https://suse-edge.github.io/charts",
				"--username",
				"user",
				"--password",
				"pass",
			},
		},
	}

	var buf bytes.Buffer

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := addRepoCommand(test.repo, &buf)

			assert.Equal(t, test.expectedArgs, cmd.Args)
			assert.Equal(t, &buf, cmd.Stdout)
			assert.Equal(t, &buf, cmd.Stderr)
		})
	}
}

func TestRegistryLoginCommand(t *testing.T) {
	tests := []struct {
		name         string
		hostURL      string
		repo         *image.HelmRepository
		expectedArgs []string
	}{
		{
			name:    "Valid Registry With Auth",
			hostURL: "registry-1.docker.io",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts/apache",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
			},
			expectedArgs: []string{
				"helm",
				"registry",
				"login",
				"registry-1.docker.io",
				"--username",
				"user",
				"--password",
				"pass",
			},
		},
	}

	var buf bytes.Buffer

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := registryLoginCommand(test.hostURL, test.repo, &buf)

			assert.Equal(t, test.expectedArgs, cmd.Args)
			assert.Equal(t, &buf, cmd.Stdout)
			assert.Equal(t, &buf, cmd.Stderr)
		})
	}
}

func TestPullCommand(t *testing.T) {
	tests := []struct {
		name         string
		repo         *image.HelmRepository
		chart        string
		version      string
		destDir      string
		expectedArgs []string
	}{
		{
			name: "OCI repository",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts/apache",
			},
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
			name: "HTTP repository",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
			},
			chart:   "kubevirt",
			version: "0.2.1",
			destDir: "charts",
			expectedArgs: []string{
				"helm",
				"pull",
				"suse-edge/kubevirt",
				"--version",
				"0.2.1",
				"--destination",
				"charts",
			},
		},
		{
			name: "OCI repository without optional args",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts/apache",
			},
			expectedArgs: []string{
				"helm",
				"pull",
				"oci://registry-1.docker.io/bitnamicharts/apache",
			},
		},
		{
			name: "HTTP repository without optional args",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
			},
			chart: "kubevirt",
			expectedArgs: []string{
				"helm",
				"pull",
				"suse-edge/kubevirt",
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
			repo:        "suse-edge/kubevirt",
			chart:       "kubevirt",
			version:     "0.2.1",
			kubeVersion: "v1.29.0+rke2r1",
			valuesPath:  "/kubevirt/values.yaml",
			expectedArgs: []string{
				"helm",
				"template",
				"--skip-crds",
				"kubevirt",
				"suse-edge/kubevirt",
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
			repo:        "suse-edge/kubevirt",
			chart:       "kubevirt",
			kubeVersion: "v1.29.0+rke2r1",
			expectedArgs: []string{
				"helm",
				"template",
				"--skip-crds",
				"kubevirt",
				"suse-edge/kubevirt",
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
