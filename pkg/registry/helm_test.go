package registry

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type mockHelm struct {
	addRepoFunc  func(chart, repository string) error
	pullFunc     func(chart, repository, version, destDir string) (string, error)
	templateFunc func(chart, repository, version, valuesFilePath, kubeVersion string, setArgs []string) ([]map[string]any, error)
}

func (m mockHelm) AddRepo(chart, repository string) error {
	if m.addRepoFunc != nil {
		return m.addRepoFunc(chart, repository)
	}
	panic("not implemented")
}

func (m mockHelm) Pull(chart, repository, version, destDir string) (string, error) {
	if m.pullFunc != nil {
		return m.pullFunc(chart, repository, version, destDir)
	}
	panic("not implemented")
}

func (m mockHelm) Template(chart, repository, version, valuesFilePath, kubeVersion string, setArgs []string) ([]map[string]any, error) {
	if m.templateFunc != nil {
		return m.templateFunc(chart, repository, version, valuesFilePath, kubeVersion, setArgs)
	}
	panic("not implemented")
}

func TestHelmCharts_EmptySourceDir(t *testing.T) {
	charts, err := HelmCharts("", "", "", nil)
	require.Error(t, err)
	assert.EqualError(t, err, "getting helm manifest paths: manifest source directory not defined")
	assert.Nil(t, charts)
}

func TestHelmCharts_MissingSourceDir(t *testing.T) {
	charts, err := HelmCharts("oops!", "", "", nil)
	require.Error(t, err)
	assert.EqualError(t, err, "getting helm manifest paths: reading manifest source dir 'oops!': open oops!: no such file or directory")
	assert.Nil(t, charts)
}

func TestHelmCharts_InvalidManifestFormat(t *testing.T) {
	dir, err := os.MkdirTemp("", "helm-chart-invalid-manifest-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()

	file := filepath.Join(dir, "invalid-format.yaml")
	require.NoError(t, os.WriteFile(file, []byte("abc"), 0o600))

	charts, err := HelmCharts(dir, "", "", nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "parsing manifest: unmarshaling resource: yaml: unmarshal errors")
	assert.ErrorContains(t, err, "line 1: cannot unmarshal !!str `abc`")
	assert.Nil(t, charts)
}

func TestHelmCharts_InvalidManifestContents(t *testing.T) {
	dir, err := os.MkdirTemp("", "helm-chart-invalid-manifest-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()

	b, err := yaml.Marshal(map[string]string{"apiVersion": "v1"})
	require.NoError(t, err)

	file := filepath.Join(dir, "invalid-crd.yaml")
	require.NoError(t, os.WriteFile(file, b, 0o600))

	charts, err := HelmCharts(dir, "", "", nil)
	require.Error(t, err)
	assert.EqualError(t, err, "resource is missing 'kind' field")
	assert.Nil(t, charts)
}

func TestHelmCharts_AddRepoError(t *testing.T) {
	dir, err := os.MkdirTemp("", "helm-charts-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()

	b, err := yaml.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "HelmChart",
		"spec": map[string]any{
			"chart": "some-chart",
			"repo":  "some-repo",
		},
	})
	require.NoError(t, err)

	file := filepath.Join(dir, "chart.yaml")
	require.NoError(t, os.WriteFile(file, b, 0o600))

	helm := mockHelm{
		addRepoFunc: func(chart, repository string) error {
			return fmt.Errorf("adding chart %s from repository %s failed", chart, repository)
		},
	}

	charts, err := HelmCharts(dir, "", "", helm)
	require.Error(t, err)
	assert.EqualError(t, err, "handling chart resource: downloading chart: adding repo: adding chart some-chart from repository some-repo failed")
	assert.Nil(t, charts)
}

func TestHelmCharts_PullError(t *testing.T) {
	dir, err := os.MkdirTemp("", "helm-charts-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()

	b, err := yaml.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "HelmChart",
		"spec": map[string]any{
			"chart": "some-chart",
			"repo":  "some-repo",
		},
	})
	require.NoError(t, err)

	file := filepath.Join(dir, "chart.yaml")
	require.NoError(t, os.WriteFile(file, b, 0o600))

	helm := mockHelm{
		addRepoFunc: func(chart, repository string) error {
			return nil
		},
		pullFunc: func(chart, repository, version, destDir string) (string, error) {
			return "", fmt.Errorf("cannot pull chart %s from repository %s", chart, repository)
		},
	}

	charts, err := HelmCharts(dir, "", "", helm)
	require.Error(t, err)
	assert.EqualError(t, err, "handling chart resource: downloading chart: pulling chart: cannot pull chart some-chart from repository some-repo")
	assert.Nil(t, charts)
}

func TestHelmCharts_TemplateError(t *testing.T) {
	dir, err := os.MkdirTemp("", "helm-charts-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()

	b, err := yaml.Marshal(map[string]any{
		"apiVersion": "v1",
		"kind":       "HelmChart",
		"spec": map[string]any{
			"chart": "some-chart",
			"repo":  "some-repo",
		},
	})
	require.NoError(t, err)

	file := filepath.Join(dir, "chart.yaml")
	require.NoError(t, os.WriteFile(file, b, 0o600))

	helm := mockHelm{
		addRepoFunc: func(chart, repository string) error {
			return nil
		},
		pullFunc: func(chart, repository, version, destDir string) (string, error) {
			return "some-path", nil
		},
		templateFunc: func(chart, repository, version, valuesFilePath, kubeVersion string, setArgs []string) ([]map[string]any, error) {
			return nil, fmt.Errorf("chart %s is invalid", chart)
		},
	}

	charts, err := HelmCharts(dir, "", "", helm)
	require.Error(t, err)
	assert.EqualError(t, err, "handling chart resource: templating chart: chart some-chart is invalid")
	assert.Nil(t, charts)
}

func writeChartFile(t *testing.T, path string, resources []map[string]any) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, file.Close())
	}()

	for _, resource := range resources {
		_, err = file.WriteString("---\n")
		require.NoError(t, err)

		b, err := yaml.Marshal(resource)
		require.NoError(t, err)

		_, err = file.Write(b)
		require.NoError(t, err)
	}
}

func TestHelmCharts(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "helm-charts-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(srcDir))
	}()

	buildDir := filepath.Join(srcDir, "build")
	require.NoError(t, os.Mkdir(buildDir, 0o700))

	encodedChart := map[string]any{
		"apiVersion": "v1",
		"kind":       "HelmChart",
		"metadata": map[string]any{
			"name": "encoded-chart",
		},
		"spec": map[string]any{
			"chartContent": "H4sIFAAAAAAA",
		},
	}

	writeChartFile(t, filepath.Join(srcDir, "encoded-chart.yaml"), []map[string]any{encodedChart})

	nonEncodedChart := []map[string]any{
		{
			"apiVersion": "v1",
			"kind":       "HelmChart",
			"spec": map[string]any{
				"chart":     "non-encoded-chart",
				"repo":      "some-repo",
				"bootstrap": false,
			},
		},
		{
			"apiVersion": "v1",
			"kind":       "Pod",
			"spec": map[string]any{
				"image": "init-image:7.1.7",
			},
		},
	}

	writeChartFile(t, filepath.Join(srcDir, "non-encoded-chart.yaml"), nonEncodedChart)

	helm := mockHelm{
		addRepoFunc: func(chart, repository string) error {
			return nil
		},
		pullFunc: func(chart, repository, version, destDir string) (string, error) {
			path := filepath.Join(buildDir, chart) + ".tgz"

			if chart == "non-encoded-chart" {
				// Simulate downloaded chart
				if fErr := os.WriteFile(path, []byte("some-content"), 0o600); fErr != nil {
					return "", fErr
				}
			}

			return path, nil
		},
		templateFunc: func(chart, repository, version, valuesFilePath, kubeVersion string, setArgs []string) ([]map[string]any, error) {
			encodedChartResources := []map[string]any{
				{
					"apiVersion": "v1",
					"kind":       "Deployment",
					"spec": map[string]any{
						"image": "deployment-image:1.2.3",
					},
				},
				{
					"apiVersion": "v1",
					"kind":       "Namespace",
					"metadata": map[string]any{
						"name": "random-ns",
					},
				},
			}

			nonEncodedChartResources := []map[string]any{
				{
					"apiVersion": "v1",
					"kind":       "Job",
					"spec": map[string]any{
						"image": "job-image:6.1.0",
					},
				},
				{
					"apiVersion": "v1",
					"kind":       "CronJob",
					"spec": map[string]any{
						"image": "cronjob-image:0.5.6",
					},
				},
			}

			if chart == "encoded-chart" {
				return encodedChartResources, nil
			}
			return nonEncodedChartResources, nil
		},
	}

	charts, err := HelmCharts(srcDir, buildDir, "", helm)
	require.NoError(t, err)
	require.Len(t, charts, 2)

	assert.Equal(t, "encoded-chart.yaml", charts[0].Filename)
	assert.Equal(t, []map[string]any{encodedChart}, charts[0].Resources)
	assert.Equal(t, []string{"deployment-image:1.2.3"}, charts[0].ContainerImages)

	assert.Equal(t, "non-encoded-chart.yaml", charts[1].Filename)
	assert.Equal(t, []map[string]any{
		{
			"apiVersion": "v1",
			"kind":       "HelmChart",
			"spec": map[string]any{
				"chartContent": base64.StdEncoding.EncodeToString([]byte("some-content")),
				"bootstrap":    false,
			},
		},
		{
			"apiVersion": "v1",
			"kind":       "Pod",
			"spec": map[string]any{
				"image": "init-image:7.1.7",
			},
		},
	}, charts[1].Resources)
	assert.ElementsMatch(t, []string{"job-image:6.1.0", "cronjob-image:0.5.6", "init-image:7.1.7"}, charts[1].ContainerImages)

	assert.FileExists(t, filepath.Join(buildDir, "encoded-chart.tgz"))
	assert.FileExists(t, filepath.Join(buildDir, "non-encoded-chart.tgz"))
}
