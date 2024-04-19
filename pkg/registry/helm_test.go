package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type mockHelmClient struct {
	addRepoFunc       func(repository *image.HelmRepository) error
	registryLoginFunc func(repository *image.HelmRepository) error
	pullFunc          func(chart string, repository *image.HelmRepository, version, destDir string) (string, error)
	templateFunc      func(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error)
}

func (m mockHelmClient) AddRepo(repository *image.HelmRepository) error {
	if m.addRepoFunc != nil {
		return m.addRepoFunc(repository)
	}
	panic("not implemented")
}

func (m mockHelmClient) RegistryLogin(repository *image.HelmRepository) error {
	if m.registryLoginFunc != nil {
		return m.registryLoginFunc(repository)
	}
	panic("not implemented")
}

func (m mockHelmClient) Pull(chart string, repository *image.HelmRepository, version, destDir string) (string, error) {
	if m.pullFunc != nil {
		return m.pullFunc(chart, repository, version, destDir)
	}
	panic("not implemented")
}

func (m mockHelmClient) Template(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error) {
	if m.templateFunc != nil {
		return m.templateFunc(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace)
	}
	panic("not implemented")
}

func TestHelmCharts_ValuesFileNotFoundError(t *testing.T) {
	helm := &image.Helm{
		Charts: []image.HelmChart{
			{
				Name:           "apache",
				RepositoryName: "apache-repo",
				Version:        "10.7.0",
				ValuesFile:     "apache-values.yaml",
			},
		},
		Repositories: []image.HelmRepository{
			{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
			},
		},
	}

	charts, err := HelmCharts(helm, "", "", "", nil)
	require.Error(t, err)
	assert.EqualError(t, err, "handling chart resource: reading values content: open apache-values.yaml: no such file or directory")
	assert.Nil(t, charts)
}

func TestHandleChart_MissingValuesDir(t *testing.T) {
	helmChart := &image.HelmChart{
		Name:           "apache",
		RepositoryName: "apache-repo",
		Version:        "10.7.0",
		ValuesFile:     "apache-values.yaml",
	}
	helmRepo := &image.HelmRepository{
		Name: "apache-repo",
		URL:  "oci://registry-1.docker.io/bitnamicharts",
	}

	chart, err := handleChart(helmChart, helmRepo, "oops!", "", "", nil)
	assert.EqualError(t, err, "reading values content: open oops!/apache-values.yaml: no such file or directory")
	assert.Nil(t, chart)
}

func TestHandleChart_FailedDownload(t *testing.T) {
	helmChart := &image.HelmChart{
		Name:           "apache",
		RepositoryName: "apache-repo",
		Version:        "10.7.0",
	}
	helmRepo := &image.HelmRepository{
		Name: "suse-edge",
		URL:  "https://suse-edge.github.io/charts",
	}

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return fmt.Errorf("failed downloading")
		},
	}

	charts, err := handleChart(helmChart, helmRepo, "", "", "", helmClient)
	require.Error(t, err)
	assert.ErrorContains(t, err, "downloading chart: adding repo: failed downloading")
	assert.Nil(t, charts)
}

func TestHandleChart_FailedTemplate(t *testing.T) {
	helmChart := &image.HelmChart{
		Name:           "apache",
		RepositoryName: "apache-repo",
		Version:        "10.7.0",
	}
	helmRepo := &image.HelmRepository{
		Name: "apache-repo",
		URL:  "oci://registry-1.docker.io/bitnamicharts",
	}

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		registryLoginFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		pullFunc: func(chart string, repository *image.HelmRepository, version, destDir string) (string, error) {
			return "", nil
		},
		templateFunc: func(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error) {
			return nil, fmt.Errorf("failed templating")
		},
	}

	charts, err := handleChart(helmChart, helmRepo, "", "", "", helmClient)
	require.Error(t, err)
	assert.ErrorContains(t, err, "templating chart: failed templating")
	assert.Nil(t, charts)
}

func TestHandleChart_FailedGetChartContent(t *testing.T) {
	helmChart := &image.HelmChart{
		Name:           "apache",
		RepositoryName: "apache-repo",
		Version:        "10.7.0",
	}
	helmRepo := &image.HelmRepository{
		Name: "apache-repo",
		URL:  "oci://registry-1.docker.io/bitnamicharts",
	}

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		registryLoginFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		pullFunc: func(chart string, repository *image.HelmRepository, version, destDir string) (string, error) {
			return "does-not-exist.tgz", nil
		},
		templateFunc: func(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error) {
			return nil, nil
		},
	}

	charts, err := handleChart(helmChart, helmRepo, "", "", "", helmClient)
	require.Error(t, err)
	assert.ErrorContains(t, err, "getting chart content: reading chart: open does-not-exist.tgz: no such file or directory")
	assert.Nil(t, charts)
}

func TestDownloadChart_FailedAddingRepo(t *testing.T) {
	helmChart := &image.HelmChart{}
	helmRepo := &image.HelmRepository{
		URL: "https://suse-edge.github.io/charts",
	}

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return fmt.Errorf("failed to add repo")
		},
	}

	chartPath, err := downloadChart(helmChart, helmRepo, helmClient, "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "adding repo: failed to add repo")
	assert.Empty(t, chartPath)
}

func TestDownloadChart_ValidRegistryLogin(t *testing.T) {
	helmChart := &image.HelmChart{}
	helmRepo := &image.HelmRepository{
		URL: "oci://registry-1.docker.io/bitnamicharts",
		Authentication: image.HelmAuthentication{
			Username: "valid",
			Password: "login",
		},
	}

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		registryLoginFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		pullFunc: func(chart string, repository *image.HelmRepository, version, destDir string) (string, error) {
			return "apache-chart.tgz", nil
		},
	}

	chartPath, err := downloadChart(helmChart, helmRepo, helmClient, "")
	require.NoError(t, err)
	assert.Equal(t, "apache-chart.tgz", chartPath)
}

func TestDownloadChart_FailedRegistryLogin(t *testing.T) {
	helmChart := &image.HelmChart{}
	helmRepo := &image.HelmRepository{
		URL: "oci://registry-1.docker.io/bitnamicharts",
		Authentication: image.HelmAuthentication{
			Username: "wrong",
			Password: "creds",
		},
	}

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		registryLoginFunc: func(repository *image.HelmRepository) error {
			return fmt.Errorf("wrong credentials")
		},
	}

	chartPath, err := downloadChart(helmChart, helmRepo, helmClient, "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "logging into registry: wrong credentials")
	assert.Empty(t, chartPath)
}

func TestDownloadChart_FailedPulling(t *testing.T) {
	helmChart := &image.HelmChart{}
	helmRepo := &image.HelmRepository{
		URL: "https://suse-edge.github.io/charts",
	}

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		pullFunc: func(chart string, repository *image.HelmRepository, version, destDir string) (string, error) {
			return "", fmt.Errorf("failed pulling chart")
		},
	}

	chartPath, err := downloadChart(helmChart, helmRepo, helmClient, "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "pulling chart: failed pulling chart")
	assert.Empty(t, chartPath)
}

func TestDownloadChart(t *testing.T) {
	helmChart := &image.HelmChart{
		Name:           "apache",
		RepositoryName: "apache-repo",
		Version:        "10.7.0",
	}
	helmRepo := &image.HelmRepository{
		Name: "apache-repo",
		URL:  "oci://registry-1.docker.io/bitnamicharts",
	}

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		pullFunc: func(chart string, repository *image.HelmRepository, version, destDir string) (string, error) {
			return "apache-chart.tgz", nil
		},
	}

	chartPath, err := downloadChart(helmChart, helmRepo, helmClient, "")
	require.NoError(t, err)
	assert.Equal(t, "apache-chart.tgz", chartPath)
}

func TestHelmCharts(t *testing.T) {
	helm := &image.Helm{
		Charts: []image.HelmChart{
			{
				Name:                  "apache",
				RepositoryName:        "apache-repo",
				Version:               "10.7.0",
				InstallationNamespace: "apache-system",
				CreateNamespace:       true,
				TargetNamespace:       "web",
			},
		},
		Repositories: []image.HelmRepository{
			{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
			},
		},
	}

	dir, err := os.MkdirTemp("", "helm-chart-charts-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(dir))
	}()

	file := filepath.Join(dir, "apache-chart.tgz")
	require.NoError(t, os.WriteFile(file, []byte("abc"), 0o600))

	helmClient := mockHelmClient{
		addRepoFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		registryLoginFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		pullFunc: func(chart string, repository *image.HelmRepository, version, destDir string) (string, error) {
			return file, nil
		},
		templateFunc: func(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error) {
			chartResource := []map[string]any{
				{
					"apiVersion": "v1",
					"kind":       "CronJob",
					"spec": map[string]any{
						"image": "cronjob-image:0.5.6",
					},
				},
				{
					"apiVersion": "v1",
					"kind":       "Job",
					"spec": map[string]any{
						"image": "job-image:6.1.0",
					},
				},
			}

			return chartResource, nil
		},
	}

	charts, err := HelmCharts(helm, "", "", "", helmClient)
	require.NoError(t, err)

	assert.ElementsMatch(t, charts[0].ContainerImages, []string{"cronjob-image:0.5.6", "job-image:6.1.0"})

	assert.Equal(t, helmChartAPIVersion, charts[0].CRD.APIVersion)
	assert.Equal(t, helmChartKind, charts[0].CRD.Kind)

	assert.Equal(t, "apache", charts[0].CRD.Metadata.Name)
	assert.Equal(t, "apache-system", charts[0].CRD.Metadata.Namespace)

	assert.Equal(t, "10.7.0", charts[0].CRD.Spec.Version)
	assert.Equal(t, "YWJj", charts[0].CRD.Spec.ChartContent)
	assert.Equal(t, "web", charts[0].CRD.Spec.TargetNamespace)
	assert.Equal(t, true, charts[0].CRD.Spec.CreateNamespace)
}

func TestMapChartRepos(t *testing.T) {
	helm := &image.Helm{
		Charts: []image.HelmChart{
			{
				Name:           "apache",
				RepositoryName: "apache-repo",
				Version:        "10.7.0",
			},
			{
				Name:           "metallb",
				RepositoryName: "suse-edge",
				Version:        "0.14.3",
			},
		},
		Repositories: []image.HelmRepository{
			{
				Name: "apache-repo",
				URL:  "oci://registry-1.docker.io/bitnamicharts",
			},
			{
				Name: "suse-edge",
				URL:  "https://suse-edge.github.io/charts",
			},
		},
	}

	expectedMap := map[string]*image.HelmRepository{
		"apache-repo": {
			Name: "apache-repo",
			URL:  "oci://registry-1.docker.io/bitnamicharts",
		},
		"suse-edge": {
			Name: "suse-edge",
			URL:  "https://suse-edge.github.io/charts",
		},
	}

	assert.True(t, reflect.DeepEqual(expectedMap, mapChartRepos(helm)))
}
