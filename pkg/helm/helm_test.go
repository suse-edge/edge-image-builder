package helm

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	certsDir = "certs"
)

func TestHelmChartPath(t *testing.T) {
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
			repoURL:        "oci://registry-1.docker.io/bitnamicharts",
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
			path := chartPath(test.repoName, test.repoURL, test.chart)
			assert.Equal(t, test.expectedOutput, path)
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
			name: "Valid repository",
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
			name: "Valid repository with auth",
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
		{
			name: "Valid repository with auth and skip TLS verify",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				SkipTLSVerify: true,
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
				"--insecure-skip-tls-verify",
			},
		},
		{
			name: "Valid repository with auth and plain HTTP",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "http://suse-edge.github.io/charts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				PlainHTTP: true,
			},
			expectedArgs: []string{
				"helm",
				"repo",
				"add",
				"suse-edge",
				"http://suse-edge.github.io/charts",
				"--username",
				"user",
				"--password",
				"pass",
			},
		},
		{
			name: "Valid repository with auth and a ca file",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				CAFile: "suse-edge.crt",
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
				"--ca-file",
				"certs/suse-edge.crt",
			},
		},
	}

	var buf bytes.Buffer

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := addRepoCommand(test.repo, certsDir, &buf)

			assert.Equal(t, test.expectedArgs, cmd.Args)
			assert.Equal(t, &buf, cmd.Stdout)
			assert.Equal(t, &buf, cmd.Stderr)
		})
	}
}

func TestRegistryLoginCommand(t *testing.T) {

	tests := []struct {
		name         string
		host         string
		repo         *image.HelmRepository
		expectedArgs []string
	}{
		{
			name: "Valid Registry With Auth",
			host: "registry-1.docker.io",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
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
		{
			name: "Valid registry with auth and skip TLS verify",
			host: "registry-1.docker.io",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				SkipTLSVerify: true,
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
				"--insecure",
			},
		},
		{
			name: "Valid registry with auth and plain HTTP",
			host: "registry-1.docker.io",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				PlainHTTP: true,
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
				"--insecure",
			},
		},
		{
			name: "Valid registry with auth and a ca file",
			host: "registry-1.docker.io",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				CAFile: "apache.crt",
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
				"--ca-file",
				"certs/apache.crt",
			},
		},
	}

	var buf bytes.Buffer

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := registryLoginCommand(test.host, test.repo, certsDir, &buf)

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
			name:  "OCI repository",
			chart: "apache",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
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
			name:  "OCI repository without optional args",
			chart: "apache",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
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
		{
			name: "HTTP repository with auth and skip TLS verify",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				SkipTLSVerify: true,
			},
			chart: "kubevirt",
			expectedArgs: []string{
				"helm",
				"pull",
				"suse-edge/kubevirt",
				"--insecure-skip-tls-verify",
			},
		},
		{
			name: "HTTP repository with auth and plain HTTP",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "http://suse-edge.github.io/charts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				PlainHTTP: true,
			},
			chart: "kubevirt",
			expectedArgs: []string{
				"helm",
				"pull",
				"suse-edge/kubevirt",
				"--plain-http",
			},
		},
		{
			name:  "OCI repository with auth and skip TLS verify",
			chart: "apache",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				SkipTLSVerify: true,
			},
			expectedArgs: []string{
				"helm",
				"pull",
				"oci://registry-1.docker.io/bitnamicharts/apache",
				"--insecure-skip-tls-verify",
			},
		},
		{
			name:  "OCI repository with auth and plain HTTP",
			chart: "apache",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				PlainHTTP: true,
			},
			expectedArgs: []string{
				"helm",
				"pull",
				"oci://registry-1.docker.io/bitnamicharts/apache",
				"--plain-http",
			},
		},
		{
			name: "HTTP repository with auth and a ca file",
			repo: &image.HelmRepository{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				CAFile: "suse-edge.crt",
			},
			chart: "kubevirt",
			expectedArgs: []string{
				"helm",
				"pull",
				"suse-edge/kubevirt",
				"--ca-file",
				"certs/suse-edge.crt",
			},
		},
		{
			name:  "OCI repository with auth and a ca file",
			chart: "apache",
			repo: &image.HelmRepository{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
				Authentication: image.HelmAuthentication{
					Username: "user",
					Password: "pass",
				},
				CAFile: "apache.crt",
			},
			expectedArgs: []string{
				"helm",
				"pull",
				"oci://registry-1.docker.io/bitnamicharts/apache",
				"--ca-file",
				"certs/apache.crt",
			},
		},
	}

	var buf bytes.Buffer

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := pullCommand(test.chart, test.repo, test.version, test.destDir, certsDir, &buf)

			assert.Equal(t, test.expectedArgs, cmd.Args)
			assert.Equal(t, &buf, cmd.Stdout)
			assert.Equal(t, &buf, cmd.Stderr)
		})
	}
}

func TestTemplateCommand(t *testing.T) {
	tests := []struct {
		name            string
		repo            string
		chart           string
		version         string
		kubeVersion     string
		targetNamespace string
		valuesPath      string
		expectedArgs    []string
	}{
		{
			name:            "Template with all parameters",
			repo:            "suse-edge/kubevirt",
			chart:           "kubevirt",
			version:         "0.2.1",
			kubeVersion:     "v1.29.0+rke2r1",
			targetNamespace: "kubevirt-ns",
			valuesPath:      "/kubevirt/values.yaml",
			expectedArgs: []string{
				"helm",
				"template",
				"--skip-crds",
				"kubevirt",
				"suse-edge/kubevirt",
				"--namespace",
				"kubevirt-ns",
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
			cmd := templateCommand(test.chart, test.repo, test.version, test.valuesPath, test.kubeVersion, test.targetNamespace, &stdout, &stderr)

			assert.Equal(t, test.expectedArgs, cmd.Args)
			assert.Equal(t, &stdout, cmd.Stdout)
			assert.Equal(t, &stderr, cmd.Stderr)
		})
	}
}

func TestParseChartContents_InvalidPayload(t *testing.T) {
	contents := `---
# Source: some-invalid.yaml
invalid-resource
`

	resources, err := parseChartContents(contents)
	require.Error(t, err)

	assert.ErrorContains(t, err, "decoding resource from source '# Source: some-invalid.yaml'")
	assert.ErrorContains(t, err, "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `invalid...` into map[string]interface {}")
	assert.Nil(t, resources)
}

func TestParseChartContents(t *testing.T) {
	contents := `
# Source: cert-manager/templates/cainjector-serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
automountServiceAccountToken: true
metadata:
  name: cert-manager-cainjector
  namespace: default
  labels:
    app: cainjector
    app.kubernetes.io/name: cainjector
    app.kubernetes.io/instance: cert-manager
    app.kubernetes.io/component: "cainjector"
    app.kubernetes.io/version: "v1.14.4"
    app.kubernetes.io/managed-by: Helm
    helm.sh/chart: cert-manager-v1.14.4
---
# Source: cert-manager/templates/serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
automountServiceAccountToken: true
metadata:
  name: cert-manager
  namespace: default
  labels:
    app: cert-manager
    app.kubernetes.io/name: cert-manager
    app.kubernetes.io/instance: cert-manager
    app.kubernetes.io/component: "controller"
    app.kubernetes.io/version: "v1.14.4"
    app.kubernetes.io/managed-by: Helm
    helm.sh/chart: cert-manager-v1.14.4
`

	resources, err := parseChartContents(contents)
	require.NoError(t, err)

	require.Len(t, resources, 2)

	assert.Equal(t, map[string]any{
		"apiVersion":                   "v1",
		"kind":                         "ServiceAccount",
		"automountServiceAccountToken": true,
		"metadata": map[string]any{
			"name":      "cert-manager-cainjector",
			"namespace": "default",
			"labels": map[string]any{
				"app":                          "cainjector",
				"app.kubernetes.io/name":       "cainjector",
				"app.kubernetes.io/instance":   "cert-manager",
				"app.kubernetes.io/component":  "cainjector",
				"app.kubernetes.io/version":    "v1.14.4",
				"app.kubernetes.io/managed-by": "Helm",
				"helm.sh/chart":                "cert-manager-v1.14.4",
			},
		},
	}, resources[0])

	assert.Equal(t, map[string]any{
		"apiVersion":                   "v1",
		"kind":                         "ServiceAccount",
		"automountServiceAccountToken": true,
		"metadata": map[string]any{
			"name":      "cert-manager",
			"namespace": "default",
			"labels": map[string]any{
				"app":                          "cert-manager",
				"app.kubernetes.io/name":       "cert-manager",
				"app.kubernetes.io/instance":   "cert-manager",
				"app.kubernetes.io/component":  "controller",
				"app.kubernetes.io/version":    "v1.14.4",
				"app.kubernetes.io/managed-by": "Helm",
				"helm.sh/chart":                "cert-manager-v1.14.4",
			},
		},
	}, resources[1])
}
