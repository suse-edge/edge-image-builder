package combustion

import (
	_ "embed"
	"fmt"
	"io"
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
	registryScriptName      = "14-embedded-registry.sh"
	registryTarName         = "embedded-registry.tar.zst"
	registryComponentName   = "embedded artifact registry"
	registryLogFileName     = "embedded-registry.log"
	hauler                  = "hauler"
	registryDir             = "registry"
	registryPort            = "6545"
	registryMirrorsFileName = "registries.yaml"

	helmLogFileName      = "helm.log"
	helmDir              = "helm"
	helmTemplateFilename = "helm.yaml"
	helmChartsDir        = "charts"
)

//go:embed templates/hauler-manifest.yaml.tpl
var haulerManifest string

//go:embed templates/14-embedded-registry.sh.tpl
var registryScript string

//go:embed templates/registries.yaml.tpl
var k8sRegistryMirrors string

func configureRegistry(ctx *image.Context) ([]string, error) {
	if !IsEmbeddedArtifactRegistryConfigured(ctx) {
		log.AuditComponentSkipped(registryComponentName)
		return nil, nil
	}

	registriesDir := filepath.Join(ctx.CombustionDir, registryDir)
	err := os.Mkdir(registriesDir, os.ModePerm)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("creating registry dir: %w", err)
	}

	var helmTemplatePath string
	var helmChartPaths []string
	if isComponentConfigured(ctx, filepath.Join(k8sDir, helmDir)) {
		helmTemplatePath = helmTemplateFilename
		helmChartPaths, err = configureHelm(ctx)
		if err != nil {
			log.AuditComponentFailed(registryComponentName)
			return nil, fmt.Errorf("configuring helm: %w", err)
		}
	}

	var localManifestSrcDir string
	if componentDir := filepath.Join(k8sDir, "manifests"); isComponentConfigured(ctx, componentDir) {
		localManifestSrcDir = filepath.Join(ctx.ImageConfigDir, componentDir)
	}

	embeddedContainerImages := ctx.ImageDefinition.EmbeddedArtifactRegistry.ContainerImages
	manifestURLs := ctx.ImageDefinition.Kubernetes.Manifests.URLs
	manifestDownloadDest := ""
	if len(manifestURLs) != 0 {
		manifestDownloadDest = filepath.Join(ctx.BuildDir, "downloaded-manifests")
		err = os.Mkdir(manifestDownloadDest, os.ModePerm)
		if err != nil {
			log.AuditComponentFailed(registryComponentName)
			return nil, fmt.Errorf("creating manifest download dir: %w", err)
		}
	}

	containerImages, err := registry.GetAllImages(embeddedContainerImages, manifestURLs, localManifestSrcDir, helmTemplatePath, manifestDownloadDest)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("getting all container images: %w", err)
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		hostnames := getImageHostnames(containerImages)

		err = writeRegistryMirrors(ctx, hostnames)
		if err != nil {
			log.AuditComponentFailed(registryComponentName)
			return nil, fmt.Errorf("writing registry mirrors: %w", err)
		}
	}

	err = writeHaulerManifest(ctx, containerImages)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing hauler manifest: %w", err)
	}

	err = syncHaulerManifest(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("populating hauler store: %w", err)
	}

	chartTars, err := addHaulerLocalCharts(ctx, helmChartPaths)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("populating hauler store with charts: %w", err)
	}
	_ = chartTars
	//fmt.Println(chartTars)
	//writeUpdatedHelmManifests(ctx, chartTars)

	err = generateRegistryTar(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("generating hauler store tar: %w", err)
	}

	haulerBinaryPath := fmt.Sprintf("hauler-%s", string(ctx.ImageDefinition.Image.Arch))
	err = copyHaulerBinary(ctx, haulerBinaryPath)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("copying hauler binary: %w", err)
	}

	chartTarPaths, err := getDownloadedCharts(helmChartPaths)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("getting downloaded helm chart paths: %w", err)
	}

	err = writeUpdatedHelmManifests(ctx, chartTarPaths)
	if err != nil {
		return nil, fmt.Errorf("writing updated helm chart manifests: %w", err)
	}

	registryScriptNameResult, err := writeRegistryScript(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("writing registry script: %w", err)
	}

	log.AuditComponentSuccessful(registryComponentName)
	return []string{registryScriptNameResult}, nil
}

func writeHaulerManifest(ctx *image.Context, images []image.ContainerImage) error {
	haulerManifestYamlFile := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	haulerDef := struct {
		ContainerImages []image.ContainerImage
	}{
		ContainerImages: images,
	}
	data, err := template.Parse(haulerManifestYamlName, haulerManifest, haulerDef)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", haulerManifestYamlName, err)
	}

	if err := os.WriteFile(haulerManifestYamlFile, []byte(data), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", haulerManifestYamlName, err)
	}

	return nil
}

func syncHaulerManifest(ctx *image.Context) error {
	haulerManifestPath := filepath.Join(ctx.BuildDir, haulerManifestYamlName)
	args := []string{"store", "sync", "--files", haulerManifestPath, "-p", "linux/amd64"}

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

func getDownloadedCharts(chartPaths []string) ([]string, error) {
	var chartTarNames []string
	for _, chart := range chartPaths {
		var expandedChart string
		if strings.Contains(chart, "*") {
			matches, err := filepath.Glob(chart)
			if err != nil {
				return nil, fmt.Errorf("error expanding wildcard %s: %w", chart, err)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no charts matched pattern: %s", chart)
			}
			expandedChart = matches[0]
			chartTarNames = append(chartTarNames, expandedChart)
		}
	}

	return chartTarNames, nil
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
	var chartsDir string
	if isComponentConfigured(ctx, filepath.Join(k8sDir, helmDir)) {
		chartsDir = helmChartsDir
	}

	version := ctx.ImageDefinition.Kubernetes.Version
	var k8sType string
	switch {
	case strings.Contains(version, image.KubernetesDistroRKE2):
		k8sType = image.KubernetesDistroRKE2
	case strings.Contains(version, image.KubernetesDistroK3S):
		k8sType = image.KubernetesDistroK3S
	}

	values := struct {
		RegistryPort        string
		RegistryDir         string
		EmbeddedRegistryTar string
		ChartsDir           string
		FileServerPort      string
		K8sType             string
	}{
		FileServerPort:      fileServerPort,
		RegistryPort:        registryPort,
		RegistryDir:         registryDir,
		EmbeddedRegistryTar: registryTarName,
		ChartsDir:           chartsDir,
		K8sType:             k8sType,
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
		len(ctx.ImageDefinition.Kubernetes.Manifests.URLs) != 0
}

func getImageHostnames(containerImages []image.ContainerImage) []string {
	var hostnames []string

	for _, containerImage := range containerImages {
		result := strings.Split(containerImage.Name, "/")
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

func createHelmCommand(ctx *image.Context, helmCommand []string) (*exec.Cmd, *os.File, error) {
	fullLogFilename := filepath.Join(ctx.BuildDir, helmLogFileName)
	logFile, err := os.OpenFile(fullLogFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening helm log file %s: %w", helmLogFileName, err)
	}

	templateFile, err := os.OpenFile(helmTemplateFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening (for append) helm template file: %w", err)
	}

	var cmd *exec.Cmd
	switch helmCommand[1] {
	case "template":
		cmd = exec.Command("helm", helmCommand...)
		multiWriter := io.MultiWriter(logFile, templateFile)
		cmd.Stdout = multiWriter
	case "pull":
		cmd = exec.Command("helm", helmCommand...)
	case "repo":
		cmd = exec.Command("helm", helmCommand...)
		cmd.Stdout = logFile
	}

	cmd.Stderr = logFile

	return cmd, logFile, nil
}

func configureHelm(ctx *image.Context) ([]string, error) {
	helmSrcDir := filepath.Join(ctx.ImageConfigDir, k8sDir, helmDir)
	helmCommands, helmChartPaths, err := registry.GenerateHelmCommands(helmSrcDir)
	if err != nil {
		return nil, fmt.Errorf("generating helm templates: %w", err)
	}

	for _, command := range helmCommands {
		err := func() error {
			commandArgs := strings.Fields(command)
			cmd, registryLog, err := createHelmCommand(ctx, commandArgs)
			if err != nil {
				return fmt.Errorf("preparing to template helm chart: %w", err)
			}
			defer func() {
				if err = registryLog.Close(); err != nil {
					zap.S().Warnf("failed to close helm log file properly: %s", err)
				}
			}()

			if err = cmd.Run(); err != nil {
				return fmt.Errorf("templating helm chart: %w: ", err)
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return helmChartPaths, nil
}

func writeUpdatedHelmManifests(ctx *image.Context, chartTars []string) error {
	helmSrcDir := filepath.Join(ctx.ImageConfigDir, k8sDir, helmDir)

	manifests, err := registry.UpdateAllManifests(helmSrcDir, chartTars)
	if err != nil {
		return fmt.Errorf("updating manifests: %w", err)
	}

	for i, manifest := range manifests {
		for j, doc := range manifest {
			data, err := yaml.Marshal(doc)
			if err != nil {
				return fmt.Errorf("marshaling data: %w", err)
			}

			fileName := fmt.Sprintf("manifest%d.yaml", i+j)
			dirPath := filepath.Join(ctx.CombustionDir, k8sDir, k8sManifestsDir)
			if err = os.MkdirAll(dirPath, os.ModePerm); err != nil {
				return fmt.Errorf("creating kubernetes manifests dir: %w", err)
			}

			filePath := filepath.Join(dirPath, fileName)
			file, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("creating manifest file: %w", err)
			}

			if _, err = file.Write(data); err != nil {
				return fmt.Errorf("writing manifest file: %w", err)
			}

			if err = file.Close(); err != nil {
				return fmt.Errorf("closing file %w", err)
			}
		}
	}

	return helmChartPaths, nil
}

//func writeUpdatedHelmManifests(ctx *image.Context, chartTars []string) {
//	helmSrcDir := filepath.Join(ctx.ImageConfigDir, k8sDir, helmDir)
//
//	stuff, err := registry.UpdateAllManifests(helmSrcDir)
//	fmt.Println("stuff", stuff)
//	fmt.Println("err", err)
//}
