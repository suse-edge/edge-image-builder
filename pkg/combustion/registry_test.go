package combustion

import (
	"fmt"
	"os"
	"path/filepath"
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
	fmt.Println(outputChartPaths)

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

				content, err := os.ReadFile(test.filePath)
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
