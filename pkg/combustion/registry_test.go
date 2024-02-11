package combustion

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/suse-edge/edge-image-builder/pkg/registry"

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
					Name:           "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
					SupplyChainKey: "carbide-key.pub",
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
	assert.Contains(t, found, "- name: rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1")
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
								Name:           "nginx",
								SupplyChainKey: "sample-key",
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
								Name:           "nginx",
								SupplyChainKey: "sample-key",
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
			Name:           "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
			SupplyChainKey: "carbide-key.pub",
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

func TestWriteStringToLog(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "temp")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()
	sampleLogPath := filepath.Join(tempDir, "sample.log")
	logFile, err := os.OpenFile(sampleLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	require.NoError(t, err)

	noPermsLogPath := filepath.Join(tempDir, "no-perms.log")
	noPermsLogFile, err := os.OpenFile(noPermsLogPath, os.O_CREATE|os.O_RDONLY, fileio.NonExecutablePerms)
	require.NoError(t, err)

	tests := []struct {
		name           string
		stringToWrite  string
		expectedString string
		file           *os.File
		filePath       string
		expectedError  string
	}{
		{
			name:           "Write String To File",
			stringToWrite:  "sample text to write",
			expectedString: "sample text to write\n",
			file:           logFile,
			filePath:       sampleLogPath,
		},
		{
			name:          "ReadOnly File",
			stringToWrite: "test",
			file:          noPermsLogFile,
			filePath:      noPermsLogPath,
			expectedError: fmt.Sprintf("writing 'test' to log file '%[1]s': write %[1]s: bad file descriptor", filepath.Join(tempDir, "no-perms.log")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = writeStringToLog(test.stringToWrite, test.file)
			if test.expectedError == "" {
				require.NoError(t, err)
				require.NoError(t, test.file.Close())

				var content []byte
				content, err = os.ReadFile(test.filePath)
				require.NoError(t, err)

				actualContent := string(content)
				assert.Equal(t, test.expectedString, actualContent)
			} else {
				require.ErrorContains(t, err, test.expectedError)

				content, err := os.ReadFile(test.filePath)
				require.NoError(t, err)

				actualContent := string(content)
				assert.Empty(t, actualContent)
			}
		})
	}
}

func TestCreateHelmCommand(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "temp")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(tempDir))
	}()
	templateLogPath := filepath.Join(tempDir, "template.log")
	templateLogFile, err := os.OpenFile(templateLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	require.NoError(t, err)

	pullLogPath := filepath.Join(tempDir, "pull.log")
	pullLogFile, err := os.OpenFile(pullLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	require.NoError(t, err)

	repoLogPath := filepath.Join(tempDir, "repo.log")
	repoLogFile, err := os.OpenFile(repoLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	require.NoError(t, err)

	readOnlyLogPath := filepath.Join(tempDir, "read-only.log")
	readOnlyLogFile, err := os.OpenFile(readOnlyLogPath, os.O_CREATE|os.O_RDONLY, fileio.NonExecutablePerms)
	require.NoError(t, err)

	logFiles := []*os.File{
		templateLogFile,
		pullLogFile,
		repoLogFile,
	}

	invalidLogFiles := []*os.File{
		readOnlyLogFile,
		readOnlyLogFile,
		readOnlyLogFile,
	}

	var helmPath string
	helmPath, err = exec.LookPath("helm")
	if err != nil {
		require.ErrorContains(t, err, "exec: \"helm\": executable file not found in $PATH")
		helmPath = "helm"
	}

	tests := []struct {
		name            string
		logFiles        []*os.File
		helmCommand     []string
		helmTemplateDir string
		expectedLog     string
		expectedFile    string
		expectedString  string
		expectedError   string
	}{
		{
			name: "Helm Template Command",
			helmCommand: []string{
				helmPath, "template", "metallb", "repo-metallb/metallb", "-f", "values.yaml",
			},
			helmTemplateDir: tempDir,
			expectedFile:    filepath.Join(tempDir, helmTemplateFilename),
			expectedLog:     templateLogPath,
			expectedString:  fmt.Sprintf("command: %s template metallb repo-metallb/metallb -f values.yaml\n", helmPath),
			logFiles:        logFiles,
		},
		{
			name: "Helm Pull Command",
			helmCommand: []string{
				helmPath, "pull", "oci://registry-1.docker.io/bitnamicharts/apache", "--version", "10.5.2",
			},
			helmTemplateDir: tempDir,
			expectedFile:    filepath.Join(tempDir, helmTemplateFilename),
			expectedLog:     pullLogPath,
			expectedString:  fmt.Sprintf("command: %s pull oci://registry-1.docker.io/bitnamicharts/apache --version 10.5.2\n", helmPath),
			logFiles:        logFiles,
		},
		{
			name: "Helm Repo Add Command",
			helmCommand: []string{
				helmPath, "repo", "add", "repo-metallb", "https://suse-edge.github.io/charts",
			},
			helmTemplateDir: tempDir,
			expectedFile:    filepath.Join(tempDir, helmTemplateFilename),
			expectedLog:     repoLogPath,
			expectedString:  fmt.Sprintf("command: %s repo add repo-metallb https://suse-edge.github.io/charts\n", helmPath),
			logFiles:        logFiles,
		},
		{
			name: "Invalid Helm Command",
			helmCommand: []string{
				"helm", "invalid",
			},
			helmTemplateDir: tempDir,
			expectedError:   "invalid helm command: 'invalid', must be 'pull', 'repo', or 'template'",
		},
		{
			name: "Template Read Only Log File",
			helmCommand: []string{
				helmPath, "template", "metallb", "repo-metallb/metallb", "-f", "values.yaml",
			},
			helmTemplateDir: tempDir,
			logFiles:        invalidLogFiles,
			expectedError:   fmt.Sprintf("writing string to log file: writing 'command: %[2]s template metallb repo-metallb/metallb -f values.yaml' to log file '%[1]s': write %[1]s: bad file descriptor", filepath.Join(tempDir, "read-only.log"), helmPath),
		},
		{
			name: "Pull Read Only Log File",
			helmCommand: []string{
				helmPath, "pull", "oci://registry-1.docker.io/bitnamicharts/apache", "--version", "10.5.2",
			},
			helmTemplateDir: tempDir,
			logFiles:        invalidLogFiles,
			expectedError:   fmt.Sprintf("writing string to log file: writing 'command: %[2]s pull oci://registry-1.docker.io/bitnamicharts/apache --version 10.5.2' to log file '%[1]s': write %[1]s: bad file descriptor", filepath.Join(tempDir, "read-only.log"), helmPath),
		},
		{
			name: "Repo Add Read Only Log File",
			helmCommand: []string{
				helmPath, "repo", "add", "repo-metallb", "https://suse-edge.github.io/charts",
			},
			helmTemplateDir: tempDir,
			logFiles:        invalidLogFiles,
			expectedError:   fmt.Sprintf("writing string to log file: writing 'command: %[2]s repo add repo-metallb https://suse-edge.github.io/charts' to log file '%[1]s': write %[1]s: bad file descriptor", filepath.Join(tempDir, "read-only.log"), helmPath),
		},
		{
			name: "Invalid Helm Template Dir",
			helmCommand: []string{
				helmPath, "repo", "add", "repo-metallb", "https://suse-edge.github.io/charts",
			},
			helmTemplateDir: "invalid",
			logFiles:        invalidLogFiles,
			expectedError:   "error opening (for append) helm template file: open invalid/helm.yaml: no such file or directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var cmd *exec.Cmd
			cmd, err = createHelmCommand(test.helmTemplateDir, test.helmCommand, test.logFiles)
			if test.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, strings.Join(test.helmCommand, " "), cmd.String())
				assert.FileExists(t, test.expectedFile)

				assert.FileExists(t, test.expectedLog)
				var content []byte
				content, err = os.ReadFile(test.expectedLog)
				require.NoError(t, err)

				actualContent := string(content)
				assert.Equal(t, test.expectedString, actualContent)
			} else {
				require.ErrorContains(t, err, test.expectedError)
			}
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

	unreadableTarPath := filepath.Join(unreadableDir, "unreadable-apache-tar.tgz")
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
			expectedError:         fmt.Sprintf("updating manifests: updating helm manifest: reading chart tar '%[1]s': open %[1]s: permission denied", filepath.Join(unreadableDir, "unreadable-apache-tar.tgz")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err = writeUpdatedHelmManifests(test.manifestDestDir, test.chartTars, test.helmManifestHolderDir, test.helmSrcDir)
			if test.expectedError == "" {
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
					assert.Contains(t, actualContent, registry.HelmChartKind)
					assert.NotContains(t, actualContent, "repo:")
					assert.NotContains(t, actualContent, "chart:")
				}
			} else {
				require.ErrorContains(t, err, test.expectedError)
			}
		})
	}
}
