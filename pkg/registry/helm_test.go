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

func TestRegistry_HelmChartImages_Empty(t *testing.T) {
	var registry Registry

	images, err := registry.helmChartImages()
	require.NoError(t, err)
	assert.Empty(t, images)
}

func TestRegistry_HelmChartImages_TemplateError(t *testing.T) {
	registry := Registry{
		helmCharts: []*helmChart{
			{
				HelmChart: image.HelmChart{
					Name:           "apache",
					RepositoryName: "apache-repo",
					Version:        "10.7.0",
				},
				repositoryURL: "oci://registry-1.docker.io/bitnamicharts",
			},
		},
		helmClient: mockHelmClient{
			templateFunc: func(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error) {
				return nil, fmt.Errorf("failed templating")
			},
		},
	}

	images, err := registry.helmChartImages()
	require.Error(t, err)
	assert.ErrorContains(t, err, "templating chart: failed templating")
	assert.Nil(t, images)
}

func TestRegistry_HelmChartImages(t *testing.T) {
	registry := Registry{
		helmCharts: []*helmChart{
			{
				HelmChart: image.HelmChart{
					Name:           "apache",
					RepositoryName: "apache-repo",
					Version:        "10.7.0",
				},
			},
		},
		helmClient: mockHelmClient{
			templateFunc: func(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error) {
				return []map[string]any{
					{
						"kind":  "Deployment",
						"image": "apache-image:1.1.1",
					},
					{
						"kind":  "Service",
						"image": "apache", // not included due to incompatible kind
					},
					{
						"kind":  "Pod",
						"image": "apache-image:1.2.3",
					},
					{
						"kind": "PersistentVolume", // missing image field
					},
				}, nil
			},
		},
	}

	images, err := registry.helmChartImages()
	require.NoError(t, err)
	assert.ElementsMatch(t, images, []string{"apache-image:1.1.1", "apache-image:1.2.3"})
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

	chartPath, err := downloadChart(helmClient, helmChart, helmRepo, "")
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
		registryLoginFunc: func(repository *image.HelmRepository) error {
			return nil
		},
		pullFunc: func(chart string, repository *image.HelmRepository, version, destDir string) (string, error) {
			return "apache-chart.tgz", nil
		},
	}

	chartPath, err := downloadChart(helmClient, helmChart, helmRepo, "")
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
		registryLoginFunc: func(repository *image.HelmRepository) error {
			return fmt.Errorf("wrong credentials")
		},
	}

	chartPath, err := downloadChart(helmClient, helmChart, helmRepo, "")
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

	chartPath, err := downloadChart(helmClient, helmChart, helmRepo, "")
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

	chartPath, err := downloadChart(helmClient, helmChart, helmRepo, "")
	require.NoError(t, err)
	assert.Equal(t, "apache-chart.tgz", chartPath)
}

func TestRegistry_HelmCharts(t *testing.T) {
	helmDir, err := os.MkdirTemp("", "helm-charts-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(helmDir))
	}()

	chartFile := filepath.Join(helmDir, "apache-chart.tgz")
	require.NoError(t, os.WriteFile(chartFile, []byte("abc"), 0o600))

	valuesFile := filepath.Join(helmDir, "apache-values.yaml")
	require.NoError(t, os.WriteFile(valuesFile, []byte("abcd"), 0o600))

	registry := Registry{
		helmCharts: []*helmChart{
			{
				HelmChart: image.HelmChart{
					Name:                  "apache",
					RepositoryName:        "apache-repo",
					Version:               "10.7.0",
					InstallationNamespace: "apache-system",
					CreateNamespace:       true,
					TargetNamespace:       "web",
					ValuesFile:            "apache-values.yaml",
				},
				localPath:     chartFile,
				repositoryURL: "oci://registry-1.docker.io/bitnamicharts",
			},
		},
		helmValuesDir: helmDir,
	}

	charts, err := registry.HelmCharts()
	require.NoError(t, err)

	assert.Equal(t, helmChartAPIVersion, charts[0].APIVersion)
	assert.Equal(t, helmChartKind, charts[0].Kind)

	assert.Equal(t, "apache", charts[0].Metadata.Name)
	assert.Equal(t, "apache-system", charts[0].Metadata.Namespace)

	assert.Equal(t, "oci://registry-1.docker.io/bitnamicharts", charts[0].Metadata.Annotations["edge.suse.com/repository-url"])
	assert.Equal(t, "edge-image-builder", charts[0].Metadata.Annotations["edge.suse.com/source"])

	assert.Equal(t, "10.7.0", charts[0].Spec.Version)
	assert.Equal(t, "YWJj", charts[0].Spec.ChartContent)
	assert.Equal(t, "web", charts[0].Spec.TargetNamespace)
	assert.Equal(t, true, charts[0].Spec.CreateNamespace)
	assert.Equal(t, "abcd", charts[0].Spec.ValuesContent)
}

func TestRegistry_HelmCharts_NonExistingChart(t *testing.T) {
	registry := Registry{
		helmCharts: []*helmChart{
			{
				HelmChart: image.HelmChart{
					Name:           "apache",
					RepositoryName: "apache-repo",
					Version:        "10.7.0",
				},
				localPath:     "does-not-exist.tgz",
				repositoryURL: "oci://registry-1.docker.io/bitnamicharts",
			},
		},
	}

	charts, err := registry.HelmCharts()
	require.Error(t, err)
	assert.EqualError(t, err, "reading chart: open does-not-exist.tgz: no such file or directory")
	assert.Nil(t, charts)
}

func TestRegistry_HelmCharts_NonExistingValues(t *testing.T) {
	helmDir, err := os.MkdirTemp("", "helm-charts-")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(helmDir))
	}()

	chartFile := filepath.Join(helmDir, "apache-chart.tgz")
	require.NoError(t, os.WriteFile(chartFile, []byte("abc"), 0o600))

	registry := Registry{
		helmCharts: []*helmChart{
			{
				HelmChart: image.HelmChart{
					Name:           "apache",
					RepositoryName: "apache-repo",
					Version:        "10.7.0",
					ValuesFile:     "does-not-exist.yaml",
				},
				localPath: chartFile,
			},
		},
		helmValuesDir: "values",
	}

	charts, err := registry.HelmCharts()
	require.Error(t, err)
	assert.EqualError(t, err, "reading values content: open values/does-not-exist.yaml: no such file or directory")
	assert.Nil(t, charts)
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

	assert.True(t, reflect.DeepEqual(expectedMap, mapChartsToRepos(helm)))
}
