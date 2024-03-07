package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/registry"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	haulerManifestYamlName  = "hauler-manifest.yaml"
	registryScriptName      = "26-embedded-registry.sh"
	registryTarName         = "embedded-registry.tar.zst"
	registryComponentName   = "embedded artifact registry"
	registryLogFileName     = "embedded-registry.log"
	hauler                  = "hauler"
	registryDir             = "registry"
	registryPort            = "6545"
	registryMirrorsFileName = "registries.yaml"

	HelmDir   = "helm"
	ValuesDir = "values"
)

//go:embed templates/hauler-manifest.yaml.tpl
var haulerManifest string

//go:embed templates/26-embedded-registry.sh.tpl
var registryScript string

//go:embed templates/registries.yaml.tpl
var k8sRegistryMirrors string

func configureRegistry(ctx *image.Context) ([]string, error) {
	if !IsEmbeddedArtifactRegistryConfigured(ctx) {
		log.AuditComponentSkipped(registryComponentName)
		return nil, nil
	}

	configured, err := configureEmbeddedArtifactRegistry(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("configuring embedded artifact registry: %w", err)
	}

	if !configured {
		log.AuditComponentSkipped(registryComponentName)
		zap.S().Info("Skipping embedded artifact registry since the provided manifests/helm charts contain no images")
		return nil, nil
	}

	registryScriptNameResult, err := writeRegistryScript(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing registry script: %w", err)
	}

	log.AuditComponentSuccessful(registryComponentName)
	return []string{registryScriptNameResult}, nil
}

func writeHaulerManifest(ctx *image.Context, images []string) error {
	haulerManifestYamlFile := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	haulerDef := struct {
		ContainerImages []string
	}{
		ContainerImages: images,
	}
	data, err := template.Parse(haulerManifestYamlName, haulerManifest, haulerDef)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", haulerManifestYamlName, err)
	}

	if err = os.WriteFile(haulerManifestYamlFile, []byte(data), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", haulerManifestYamlName, err)
	}

	return nil
}

func syncHaulerManifest(ctx *image.Context) error {
	haulerManifestPath := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	args := []string{"store", "sync", "--files", haulerManifestPath, "-p", fmt.Sprintf("linux/%s", ctx.ImageDefinition.Image.Arch.Short())}

	cmd, registryLog, err := createRegistryCommand(ctx, hauler, args)
	if err != nil {
		return fmt.Errorf("preparing to populate registry store: %w", err)
	}
	defer func() {
		if err = registryLog.Close(); err != nil {
			zap.S().Warnf("failed to close registry log file properly: %s", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("populating hauler store: %w: ", err)
	}

	return nil
}

func generateRegistryTar(ctx *image.Context) error {
	haulerTarDest := filepath.Join(ctx.CombustionDir, registryDir, registryTarName)
	args := []string{"store", "save", "--filename", haulerTarDest}

	cmd, registryLog, err := createRegistryCommand(ctx, hauler, args)
	if err != nil {
		return fmt.Errorf("preparing to generate registry tar: %w", err)
	}
	defer func() {
		if err = registryLog.Close(); err != nil {
			zap.S().Warnf("failed to close registry log file properly: %s", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("creating registry tar: %w: ", err)
	}

	return nil
}

func copyHaulerBinary(ctx *image.Context, haulerBinaryPath string) error {
	destinationDir := filepath.Join(ctx.CombustionDir, "hauler")

	err := fileio.CopyFile(haulerBinaryPath, destinationDir, fileio.ExecutablePerms)
	if err != nil {
		return fmt.Errorf("copying hauler binary to combustion dir: %w", err)
	}

	return nil
}

func writeRegistryScript(ctx *image.Context) (string, error) {
	values := struct {
		RegistryPort        string
		RegistryDir         string
		EmbeddedRegistryTar string
	}{
		RegistryPort:        registryPort,
		RegistryDir:         registryDir,
		EmbeddedRegistryTar: registryTarName,
	}

	data, err := template.Parse(registryScriptName, registryScript, &values)
	if err != nil {
		return "", fmt.Errorf("parsing registry script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, registryScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		return "", fmt.Errorf("writing registry script: %w", err)
	}

	return registryScriptName, nil
}

func createRegistryCommand(ctx *image.Context, commandName string, args []string) (*exec.Cmd, *os.File, error) {
	fullLogFilename := filepath.Join(ctx.BuildDir, registryLogFileName)
	logFile, err := os.OpenFile(fullLogFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening registry log file %s: %w", registryLogFileName, err)
	}

	cmd := exec.Command(commandName, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd, logFile, nil
}

func IsEmbeddedArtifactRegistryConfigured(ctx *image.Context) bool {
	return len(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages) != 0 ||
		len(ctx.ImageDefinition.Kubernetes.Manifests.URLs) != 0 ||
		len(ctx.ImageDefinition.Kubernetes.HelmCharts) != 0 ||
		isComponentConfigured(ctx, filepath.Join(K8sDir, k8sManifestsDir))
}

func getImageHostnames(containerImages []string) []string {
	var hostnames []string

	for _, containerImage := range containerImages {
		result := strings.Split(containerImage, "/")
		if len(result) > 1 {
			if !slices.Contains(hostnames, result[0]) && result[0] != "docker.io" {
				hostnames = append(hostnames, result[0])
			}
		}
	}

	return hostnames
}

func writeRegistryMirrors(ctx *image.Context, hostnames []string) error {
	registriesYamlFile := filepath.Join(ctx.CombustionDir, registryMirrorsFileName)
	registriesDef := struct {
		Hostnames []string
		Port      string
	}{
		Hostnames: hostnames,
		Port:      registryPort,
	}

	data, err := template.Parse(registryMirrorsFileName, k8sRegistryMirrors, registriesDef)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", registryMirrorsFileName, err)
	}

	if err := os.WriteFile(registriesYamlFile, []byte(data), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", registryMirrorsFileName, err)
	}

	return nil
}

func configureEmbeddedArtifactRegistry(ctx *image.Context) (bool, error) {
	helmCharts, err := parseHelmCharts(ctx)
	if err != nil {
		return false, fmt.Errorf("parsing helm charts: %w", err)
	}

	if err = storeHelmCharts(ctx, helmCharts); err != nil {
		return false, fmt.Errorf("storing helm charts: %w", err)
	}

	manifestImages, err := parseManifests(ctx)
	if err != nil {
		return false, fmt.Errorf("parsing manifests: %w", err)
	}

	images := containerImages(ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages, manifestImages, helmCharts)
	if len(images) == 0 {
		return false, nil
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		hostnames := getImageHostnames(images)

		err = writeRegistryMirrors(ctx, hostnames)
		if err != nil {
			return false, fmt.Errorf("writing registry mirrors: %w", err)
		}
	}

	registriesDir := filepath.Join(ctx.CombustionDir, registryDir)
	if err = os.Mkdir(registriesDir, os.ModePerm); err != nil {
		return false, fmt.Errorf("creating registry dir: %w", err)
	}

	if err = writeHaulerManifest(ctx, images); err != nil {
		return false, fmt.Errorf("writing hauler manifest: %w", err)
	}

	if err = syncHaulerManifest(ctx); err != nil {
		return false, fmt.Errorf("populating hauler store: %w", err)
	}

	if err = generateRegistryTar(ctx); err != nil {
		return false, fmt.Errorf("generating hauler store tar: %w", err)
	}

	haulerBinaryPath := "/usr/bin/hauler"
	if err = copyHaulerBinary(ctx, haulerBinaryPath); err != nil {
		return false, fmt.Errorf("copying hauler binary: %w", err)
	}

	return true, nil
}

func containerImages(embeddedImages []image.ContainerImage, manifestImages []string, helmCharts []*registry.HelmChart) []string {
	imageSet := map[string]bool{}

	for _, img := range embeddedImages {
		imageSet[img.Name] = true
	}

	for _, img := range manifestImages {
		imageSet[img] = true
	}

	for _, chart := range helmCharts {
		for _, img := range chart.ContainerImages {
			imageSet[img] = true
		}
	}

	var images []string

	for img := range imageSet {
		images = append(images, img)
	}

	return images
}

func parseManifests(ctx *image.Context) ([]string, error) {
	var manifestSrcDir string
	if componentDir := filepath.Join(K8sDir, k8sManifestsDir); isComponentConfigured(ctx, componentDir) {
		manifestSrcDir = filepath.Join(ctx.ImageConfigDir, componentDir)
	}

	if manifestSrcDir != "" && ctx.ImageDefinition.Kubernetes.Version == "" {
		return nil, fmt.Errorf("kubernetes manifests are provided but kubernetes version is not configured")
	}

	return registry.ManifestImages(ctx.ImageDefinition.Kubernetes.Manifests.URLs, manifestSrcDir)
}

func parseHelmCharts(ctx *image.Context) ([]*registry.HelmChart, error) {
	if len(ctx.ImageDefinition.Kubernetes.HelmCharts) == 0 {
		return nil, nil
	}

	if ctx.ImageDefinition.Kubernetes.Version == "" {
		return nil, fmt.Errorf("helm charts are provided but kubernetes version is not configured")
	}

	buildDir := filepath.Join(ctx.BuildDir, HelmDir)
	if err := os.MkdirAll(buildDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating helm dir: %w", err)
	}

	helmValuesDir := filepath.Join(ctx.ImageConfigDir, K8sDir, HelmDir, ValuesDir)

	return registry.HelmCharts(ctx.ImageDefinition.Kubernetes.HelmCharts, helmValuesDir, buildDir, ctx.ImageDefinition.Kubernetes.Version, ctx.Helm)
}

func storeHelmCharts(ctx *image.Context, helmCharts []*registry.HelmChart) error {
	if len(helmCharts) == 0 {
		return nil
	}

	manifestsDir := filepath.Join(ctx.CombustionDir, K8sDir, k8sManifestsDir)
	if err := os.MkdirAll(manifestsDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating kubernetes manifests dir: %w", err)
	}

	for _, chart := range helmCharts {
		data, err := yaml.Marshal(chart.CRD)
		if err != nil {
			return fmt.Errorf("marshaling resource: %w", err)
		}

		chartFileName := fmt.Sprintf("%s.yaml", chart.CRD.Metadata.Name)
		if err = os.WriteFile(filepath.Join(manifestsDir, chartFileName), data, fileio.NonExecutablePerms); err != nil {
			return fmt.Errorf("storing manifest '%s: %w", chartFileName, err)
		}
	}

	return nil
}
