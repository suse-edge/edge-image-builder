package combustion

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestWriteHaulerManifestValidManifest(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
			ContainerImages: []image.ContainerImage{
				{
					Name: "hello-world:latest",
				},
				{
					Name: "ghcr.io/fluxcd/flux-cli@sha256:02aa820c3a9c57d67208afcfc4bce9661658c17d15940aea369da259d2b976dd",
				},
			},
		},
	}

	// Test
	err := writeHaulerManifest(ctx, ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages)

	// Verify
	require.NoError(t, err)

	manifestFileName := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	_, err = os.Stat(manifestFileName)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(manifestFileName)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, "- name: hello-world:latest")
	assert.Contains(t, found, "- name: ghcr.io/fluxcd/flux-cli@sha256:02aa820c3a9c57d67208afcfc4bce9661658c17d15940aea369da259d2b976dd")
}

func TestCreateRegistryCommand(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	cmd, logFile, err := createRegistryCommand(ctx, "testName", []string{"--flag", "test"})

	// Verify
	require.NoError(t, err)
	require.NotNil(t, cmd)

	expectedCommand := "testName"
	expectedArgs := []string{"testName", "--flag", "test"}

	assert.Equal(t, expectedCommand, cmd.Path)
	assert.Equal(t, expectedArgs, cmd.Args)

	assert.Equal(t, logFile, cmd.Stdout)
	assert.Equal(t, logFile, cmd.Stderr)

	foundFile := filepath.Join(ctx.BuildDir, "embedded-registry.log")
	_, err = os.ReadFile(foundFile)
	require.NoError(t, err)
}

func TestWriteRegistryScript(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	_, err := writeRegistryScript(ctx)

	// Verify
	require.NoError(t, err)

	registryScriptPath := filepath.Join(ctx.CombustionDir, registryScriptName)
	_, err = os.Stat(registryScriptPath)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(registryScriptPath)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, registryDir)
	assert.Contains(t, found, registryPort)
	assert.Contains(t, found, registryTarName)
	assert.Contains(t, found, "mv hauler /usr/local/bin/hauler")
	assert.Contains(t, found, "systemctl enable eib-embedded-registry.service")
	assert.Contains(t, found, "ExecStartPre=/usr/local/bin/hauler store load")
}

func TestCopyHaulerBinary(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	haulerBinaryPath := filepath.Join(ctx.BuildDir, fmt.Sprintf("hauler-%s", string(ctx.ImageDefinition.Image.Arch)))
	err := os.WriteFile(haulerBinaryPath, []byte(""), fileio.ExecutablePerms)
	require.NoError(t, err)

	// Test
	err = copyHaulerBinary(ctx, haulerBinaryPath)

	// Verify
	require.NoError(t, err)

	haulerPath := filepath.Join(ctx.CombustionDir, "hauler")
	_, err = os.Stat(haulerPath)
	require.NoError(t, err)
}

func TestCopyHaulerBinaryNoFile(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	err := copyHaulerBinary(ctx, "")

	// Verify
	require.ErrorContains(t, err, "no such file")
}

func TestIsEmbeddedArtifactRegistryConfigured(t *testing.T) {
	tests := []struct {
		name         string
		ctx          *image.Context
		isConfigured bool
	}{
		{
			name: "Everything Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
						ContainerImages: []image.ContainerImage{
							{
								Name: "nginx",
							},
						},
					},
					Kubernetes: image.Kubernetes{
						Manifests: image.Manifests{
							URLs: []string{
								"https://k8s.io/examples/application/nginx-app.yaml",
							},
						},
					},
				},
			},
			isConfigured: true,
		},
		{
			name: "Image Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{
						ContainerImages: []image.ContainerImage{
							{
								Name: "nginx",
							},
						},
					},
				},
			},
			isConfigured: true,
		},
		{
			name: "Manifest URL Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					Kubernetes: image.Kubernetes{
						Manifests: image.Manifests{
							URLs: []string{
								"https://k8s.io/examples/application/nginx-app.yaml",
							},
						},
					},
				},
			},
			isConfigured: true,
		},
		{
			name: "None Defined",
			ctx: &image.Context{
				ImageDefinition: &image.Definition{
					EmbeddedArtifactRegistry: image.EmbeddedArtifactRegistry{},
					Kubernetes:               image.Kubernetes{},
				},
			},
			isConfigured: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsEmbeddedArtifactRegistryConfigured(test.ctx)
			assert.Equal(t, test.isConfigured, result)
		})
	}
}

func TestWriteRegistryMirrorsValid(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	hostnames := []string{"hello-world:latest", "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1", "quay.io"}

	// Test
	err := writeRegistryMirrors(ctx, hostnames)

	// Verify
	require.NoError(t, err)

	manifestFileName := filepath.Join(ctx.CombustionDir, registryMirrorsFileName)
	_, err = os.Stat(manifestFileName)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(manifestFileName)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, "- \"http://localhost:6545\"")
	assert.Contains(t, found, "docker.io")
	assert.Contains(t, found, "rgcrprod.azurecr.us")
	assert.Contains(t, found, "quay.io")
}

func TestGetImageHostnames(t *testing.T) {
	// Setup
	containerImages := []image.ContainerImage{
		{
			Name: "hello-world:latest",
		},
		{
			Name: "quay.io/podman/hello",
		},
		{
			Name: "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
		},
	}
	expectedHostnames := []string{"quay.io", "rgcrprod.azurecr.us"}

	// Test
	hostnames := getImageHostnames(containerImages)

	// Verify
	assert.Equal(t, expectedHostnames, hostnames)
}

func TestGetDownloadedCharts(t *testing.T) {
	// Setup
	tempDir, err := os.MkdirTemp("", "temp")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()
	sampleChartPath := filepath.Join(tempDir, "apache-10.5.2.tgz")
	err = os.WriteFile(sampleChartPath, []byte(""), fileio.NonExecutablePerms)
	require.NoError(t, err)

	sampleChartPath2 := filepath.Join(tempDir, "metallb-1.0.0.tgz")
	err = os.WriteFile(sampleChartPath2, []byte(""), fileio.NonExecutablePerms)
	require.NoError(t, err)

	chartPaths := []string{
		filepath.Join(tempDir, "metallb-*.tgz"),
		filepath.Join(tempDir, "apache-*.tgz"),
	}

	expectedChartPaths := []string{
		sampleChartPath,
		sampleChartPath2,
	}

	// Test
	outputChartPaths, err := getDownloadedCharts(chartPaths)

	// Verify
	require.NoError(t, err)
	assert.ElementsMatch(t, expectedChartPaths, outputChartPaths)
}

func TestGetDownloadedChartsNoWildCard(t *testing.T) {
	// Setup
	chartPaths := []string{
		"metallb.tgz",
		"apache.tgz",
	}

	// Test
	outputChartPaths, err := getDownloadedCharts(chartPaths)

	// Verify
	require.NoError(t, err)
	assert.Empty(t, outputChartPaths)
}

func TestGetDownloadedChartsMalformedPattern(t *testing.T) {
	// Setup
	chartPaths := []string{
		filepath.Join("", "[metallb.tgz-*"),
	}
	expectedError := "error expanding wildcard [metallb.tgz-*: syntax error in pattern"

	// Test
	_, err := getDownloadedCharts(chartPaths)

	// Verify
	require.ErrorContains(t, err, expectedError)
}

func TestGetDownloadedChartsNoDir(t *testing.T) {
	// Setup
	chartPaths := []string{
		"metallb-*.tgz",
		"apache-*.tgz",
	}
	expectedError := "no charts matched pattern: metallb-*.tgz"

	// Test
	outputChartPaths, err := getDownloadedCharts(chartPaths)

	// Verify
	require.ErrorContains(t, err, expectedError)
	assert.Empty(t, outputChartPaths)
}

func TestCreateHelmCommand(t *testing.T) {
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		require.ErrorContains(t, err, "exec: \"helm\": executable file not found in $PATH")
		helmPath = "helm"
	}

	tests := []struct {
		name           string
		helmCommand    []string
		expectedString string
	}{
		{
			name: "Helm Template Command",
			helmCommand: []string{
				helmPath, "template", "metallb", "repo-metallb/metallb", "-f", "values.yaml",
			},
			expectedString: fmt.Sprintf("command: %s template metallb repo-metallb/metallb -f values.yaml\n", helmPath),
		},
		{
			name: "Helm Pull Command",
			helmCommand: []string{
				helmPath, "pull", "oci://registry-1.docker.io/bitnamicharts/apache", "--version", "10.5.2",
			},
			expectedString: fmt.Sprintf("command: %s pull oci://registry-1.docker.io/bitnamicharts/apache --version 10.5.2\n", helmPath),
		},
		{
			name: "Helm Repo Add Command",
			helmCommand: []string{
				helmPath, "repo", "add", "repo-metallb", "https://suse-edge.github.io/charts",
			},
			expectedString: fmt.Sprintf("command: %s repo add repo-metallb https://suse-edge.github.io/charts\n", helmPath),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			cmd, err := createHelmCommand(test.helmCommand, &stdout, &stderr)
			require.NoError(t, err)

			assert.Equal(t, strings.Join(test.helmCommand, " "), cmd.String())
			assert.Equal(t, &stdout, cmd.Stdout)
			assert.Equal(t, &stderr, cmd.Stderr)
			assert.Equal(t, test.expectedString, stdout.String())
		})
	}
}

func dirEntriesToNames(entries []os.DirEntry) []string {
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names
}

func TestWriteUpdatedHelmManifests(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "temp")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()

	unreadableDir := filepath.Join(tempDir, "unreadable")
	err = os.Mkdir(unreadableDir, os.ModePerm)
	require.NoError(t, err)

	unreadableManifestPath := filepath.Join(unreadableDir, "unreadable-manifest.yaml")
	err = os.WriteFile(unreadableManifestPath, []byte(""), 0o00)
	require.NoError(t, err)

	unreadableTarPath := filepath.Join(unreadableDir, "unreadable-metallb-tar.tgz")
	err = os.WriteFile(unreadableTarPath, []byte(""), 0o00)
	require.NoError(t, err)

	validHelmSrcDir := filepath.Join("..", "registry", "testdata", "helm", "valid")
	invalidHelmSrcDir := filepath.Join("..", "registry", "testdata", "helm", "invalid")

	manifestHolderDir := filepath.Join(tempDir, helmManifestHolderDirName)
	err = os.Mkdir(manifestHolderDir, os.ModePerm)
	require.NoError(t, err)

	k8sManifestsDestDir := filepath.Join(tempDir, k8sDir, k8sManifestsDir)
	err = os.MkdirAll(k8sManifestsDestDir, os.ModePerm)
	require.NoError(t, err)

	validChartTars := []string{
		filepath.Join("..", "registry", "testdata", "helm", "apache-10.5.2.tgz"),
		filepath.Join("..", "registry", "testdata", "helm", "metallb.tgz"),
	}

	tests := []struct {
		name                  string
		helmSrcDir            string
		helmManifestHolderDir string
		manifestDestDir       string
		chartTars             []string
		expectedManifests     []string
		expectedError         string
	}{
		{
			name:                  "Write Updated Helm Manifests",
			helmSrcDir:            validHelmSrcDir,
			manifestDestDir:       k8sManifestsDestDir,
			helmManifestHolderDir: manifestHolderDir,
			expectedManifests: []string{
				"manifest-0.yaml",
				"manifest-1.yaml",
				"manifest-2.yaml",
			},
			chartTars: validChartTars,
		},
		{
			name:          "Invalid Helm Source Dir",
			helmSrcDir:    invalidHelmSrcDir,
			expectedError: "updating manifests: updating helm manifest: unmarshaling manifest '../registry/testdata/helm/invalid/invalid.yaml': yaml: line 3: mapping values are not allowed in this context",
		},
		{
			name:                  "Invalid Manifest Holder Dir",
			helmSrcDir:            validHelmSrcDir,
			chartTars:             validChartTars,
			helmManifestHolderDir: "invalid-holder",
			expectedError:         "writing manifest file to manifest holder: open invalid-holder/manifest-0.yaml: no such file or directory",
		},
		{
			name:                  "Invalid K8s Manifest Dest Dir",
			helmSrcDir:            validHelmSrcDir,
			chartTars:             validChartTars,
			helmManifestHolderDir: manifestHolderDir,
			manifestDestDir:       "invalid-dest",
			expectedError:         "writing manifest file to combustion destination: open invalid-dest/manifest-0.yaml: no such file or directory",
		},
		{
			name:                  "Unreadable Manifest",
			helmSrcDir:            unreadableDir,
			chartTars:             validChartTars,
			helmManifestHolderDir: manifestHolderDir,
			manifestDestDir:       k8sManifestsDestDir,
			expectedError:         fmt.Sprintf("updating manifests: updating helm manifest: reading helm manifest '%[1]s': open %[1]s: permission denied", filepath.Join(unreadableDir, "unreadable-manifest.yaml")),
		},
		{
			name:       "Unreadable Chart Tar",
			helmSrcDir: validHelmSrcDir,
			chartTars: []string{
				unreadableTarPath,
			},
			helmManifestHolderDir: manifestHolderDir,
			manifestDestDir:       k8sManifestsDestDir,
			expectedError:         fmt.Sprintf("updating manifests: updating helm manifest: reading chart tar '%[1]s': open %[1]s: permission denied", filepath.Join(unreadableDir, "unreadable-metallb-tar.tgz")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = writeUpdatedHelmManifests(test.manifestDestDir, test.chartTars, test.helmManifestHolderDir, test.helmSrcDir)
			if test.expectedError != "" {
				require.ErrorContains(t, err, test.expectedError)
				return
			}

			require.NoError(t, err)

			var manifestHolderDirContents []fs.DirEntry
			manifestHolderDirContents, err = os.ReadDir(test.helmManifestHolderDir)
			require.NoError(t, err)
			actualManifests := dirEntriesToNames(manifestHolderDirContents)
			assert.ElementsMatch(t, test.expectedManifests, actualManifests)

			var manifestDestDirContents []fs.DirEntry
			manifestDestDirContents, err = os.ReadDir(test.manifestDestDir)
			require.NoError(t, err)
			actualManifests = dirEntriesToNames(manifestDestDirContents)
			assert.ElementsMatch(t, test.expectedManifests, actualManifests)

			for _, manifestName := range actualManifests {
				var manifestContent []byte
				manifestContent, err = os.ReadFile(filepath.Join(test.manifestDestDir, manifestName))
				require.NoError(t, err)

				actualContent := string(manifestContent)
				assert.Contains(t, actualContent, "chartContent:")
				assert.Contains(t, actualContent, "HelmChart")
				assert.NotContains(t, actualContent, "repo:")
				assert.NotContains(t, actualContent, "chart:")
			}
		})
	}
}
