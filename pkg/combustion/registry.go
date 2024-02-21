package combustion

import (
	"bytes"
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
	registryScriptName      = "26-embedded-registry.sh"
	registryTarName         = "embedded-registry.tar.zst"
	registryComponentName   = "embedded artifact registry"
	registryLogFileName     = "embedded-registry.log"
	hauler                  = "hauler"
	registryDir             = "registry"
	registryPort            = "6545"
	registryMirrorsFileName = "registries.yaml"

	templateLogFileName       = "helm-template.log"
	pullLogFileName           = "helm-pull.log"
	repoAddLogFileName        = "helm-repo-add.log"
	helmDir                   = "helm"
	helmTemplateFilename      = "helm.yaml"
	helmManifestHolderDirName = "manifest-holder"
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

	helmTemplatePath, helmManifestHolderDir, err := configureHelm(ctx)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("configuring helm: %w", err)
	}

	configured, err := configureEmbeddedArtifactRegistry(ctx, helmTemplatePath, helmManifestHolderDir)
	if err != nil {
		log.AuditComponentFailed(registryComponentName)
		return nil, fmt.Errorf("configuring embedded artifact registry: %w", err)
	}

	if !configured {
		log.AuditComponentSkipped(registryComponentName)
		log.Audit("Embedded Artifact Registry skipped: Provided manifests/helm charts contain no images.")
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

func getDownloadedCharts(chartPaths []string) ([]string, error) {
	var chartTarNames []string
	for _, chart := range chartPaths {
		if !strings.Contains(chart, "*") {
			continue
		}

		matches, err := filepath.Glob(chart)
		if err != nil {
			return nil, fmt.Errorf("error expanding wildcard %s: %w", chart, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no charts matched pattern: %s", chart)
		}
		expandedChart := matches[0]
		chartTarNames = append(chartTarNames, expandedChart)
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
		isComponentConfigured(ctx, filepath.Join(k8sDir, helmDir))
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

func createHelmCommand(helmCommand []string, stdout, stderr io.Writer) (*exec.Cmd, error) {
	const commandLogPrefix = "command: "

	cmd := exec.Command("helm")
	cmd.Args = helmCommand
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if _, err := fmt.Fprintln(stdout, commandLogPrefix+cmd.String()); err != nil {
		return nil, fmt.Errorf("writing command prefix to log file: %w", err)
	}

	return cmd, nil
}

func configureHelmCommands(ctx *image.Context, helmDestDir string) ([]string, error) {
	helmSrcDir := filepath.Join(ctx.ImageConfigDir, k8sDir, helmDir)
	helmCommands, helmChartPaths, err := registry.GenerateHelmCommands(helmSrcDir, helmDestDir)
	if err != nil {
		return nil, fmt.Errorf("generating helm commands: %w", err)
	}

	templateLogFilePath := filepath.Join(ctx.BuildDir, templateLogFileName)
	templateLogFile, err := os.OpenFile(templateLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, fmt.Errorf("opening helm template log file %s: %w", templateLogFilePath, err)
	}
	defer func() {
		if err = templateLogFile.Close(); err != nil {
			zap.S().Warnf("failed to close helm template log file properly: %s", err)
		}
	}()

	pullLogFilePath := filepath.Join(ctx.BuildDir, pullLogFileName)
	pullLogFile, err := os.OpenFile(pullLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, fmt.Errorf("opening helm pull log file %s: %w", pullLogFilePath, err)
	}
	defer func() {
		if err = pullLogFile.Close(); err != nil {
			zap.S().Warnf("failed to close helm pull log file properly: %s", err)
		}
	}()

	repoAddLogFilePath := filepath.Join(ctx.BuildDir, repoAddLogFileName)
	repoAddLogFile, err := os.OpenFile(repoAddLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
	if err != nil {
		return nil, fmt.Errorf("opening helm repo add log file %s: %w", repoAddLogFilePath, err)
	}
	defer func() {
		if err = repoAddLogFile.Close(); err != nil {
			zap.S().Warnf("failed to close helm repo add log file properly: %s", err)
		}
	}()

	for _, command := range helmCommands {
		err = executeHelmCommand(helmDestDir, command, templateLogFile, pullLogFile, repoAddLogFile)
		if err != nil {
			return nil, fmt.Errorf("executing helm command: %w", err)
		}
	}

	return helmChartPaths, nil
}

func executeHelmCommand(templateDir string, command string, templateLogFile, pullLogFile, repoAddLogFile *os.File) error {
	commandArgs := strings.Fields(command)

	var stdout, stderr io.Writer

	switch subcommand := commandArgs[1]; subcommand {
	case "template":
		templatePath := filepath.Join(templateDir, helmTemplateFilename)
		templateFile, err := os.OpenFile(templatePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fileio.NonExecutablePerms)
		if err != nil {
			return fmt.Errorf("error opening (for append) helm template file: %w", err)
		}
		defer templateFile.Close()

		stdout = io.MultiWriter(templateLogFile, templateFile)
		stderr = templateLogFile
	case "pull":
		stdout = pullLogFile
		stderr = pullLogFile
	case "repo":
		stdout = repoAddLogFile
		stderr = repoAddLogFile
	default:
		return fmt.Errorf("invalid helm command: '%s', must be 'pull', 'repo', or 'template'", subcommand)
	}

	cmd, err := createHelmCommand(commandArgs, stdout, stderr)
	if err != nil {
		return fmt.Errorf("creating helm command: %w", err)
	}

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("running 'helm %s': %w", commandArgs[1], err)
	}

	return nil
}

func writeUpdatedHelmManifests(k8sManifestsDir string, chartTars []string, manifestsHolderDir string, helmSrcDir string) error {
	manifests, err := registry.UpdateHelmManifests(helmSrcDir, chartTars)
	if err != nil {
		return fmt.Errorf("updating manifests: %w", err)
	}

	var buf bytes.Buffer

	for i, manifest := range manifests {
		for _, doc := range manifest {
			buf.WriteString("---\n")

			var data []byte
			data, err = yaml.Marshal(doc)
			if err != nil {
				return fmt.Errorf("marshaling data: %w", err)
			}

			buf.Write(data)
		}

		b := buf.Bytes()

		fileName := fmt.Sprintf("manifest-%d.yaml", i)
		filePath := filepath.Join(manifestsHolderDir, fileName)
		if err = os.WriteFile(filePath, b, fileio.NonExecutablePerms); err != nil {
			return fmt.Errorf("writing manifest file to manifest holder: %w", err)
		}

		destFilePath := filepath.Join(k8sManifestsDir, fileName)
		if err = os.WriteFile(destFilePath, b, fileio.NonExecutablePerms); err != nil {
			return fmt.Errorf("writing manifest file to combustion destination: %w", err)
		}
	}

	return nil
}

func configureEmbeddedArtifactRegistry(ctx *image.Context, helmTemplatePath string, helmManifestHolderDir string) (bool, error) {
	registriesDir := filepath.Join(ctx.CombustionDir, registryDir)
	err := os.Mkdir(registriesDir, os.ModePerm)
	if err != nil {
		return false, fmt.Errorf("creating registry dir: %w", err)
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
			return false, fmt.Errorf("creating manifest download dir: %w", err)
		}
	}

	containerImages, err := registry.GetAllImages(embeddedContainerImages, manifestURLs, localManifestSrcDir, helmManifestHolderDir, helmTemplatePath, manifestDownloadDest)
	if err != nil {
		return false, fmt.Errorf("getting all container images: %w", err)
	}

	if len(containerImages) == 0 {
		return false, nil
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		hostnames := getImageHostnames(containerImages)

		err = writeRegistryMirrors(ctx, hostnames)
		if err != nil {
			return false, fmt.Errorf("writing registry mirrors: %w", err)
		}
	}

	err = writeHaulerManifest(ctx, containerImages)
	if err != nil {
		return false, fmt.Errorf("writing hauler manifest: %w", err)
	}

	err = syncHaulerManifest(ctx)
	if err != nil {
		return false, fmt.Errorf("populating hauler store: %w", err)
	}

	err = generateRegistryTar(ctx)
	if err != nil {
		return false, fmt.Errorf("generating hauler store tar: %w", err)
	}

	haulerBinaryPath := fmt.Sprintf("hauler-%s", string(ctx.ImageDefinition.Image.Arch))
	err = copyHaulerBinary(ctx, haulerBinaryPath)
	if err != nil {
		return false, fmt.Errorf("copying hauler binary: %w", err)
	}

	return true, nil
}

func configureHelm(ctx *image.Context) (helmTemplatePath string, helmManifestHolderDir string, err error) {
	if !isComponentConfigured(ctx, filepath.Join(k8sDir, helmDir)) {
		return "", "", nil
	}

	var helmChartPaths []string
	var k8sManifestsDestDir string

	helmBuildDir := filepath.Join(ctx.BuildDir, helmDir)
	if err = os.Mkdir(helmBuildDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating helm build directory: %w", err)
	}

	helmTemplatePath = filepath.Join(helmBuildDir, helmTemplateFilename)
	helmChartPaths, err = configureHelmCommands(ctx, helmBuildDir)
	if err != nil {
		return "", "", fmt.Errorf("configuring helm commands: %w", err)
	}

	helmManifestHolderDir = filepath.Join(helmBuildDir, helmManifestHolderDirName)
	if err = os.Mkdir(helmManifestHolderDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating manifest holder dir: %w", err)
	}

	k8sManifestsDestDir = filepath.Join(ctx.CombustionDir, k8sDir, k8sManifestsDir)
	if err = os.MkdirAll(k8sManifestsDestDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating kubernetes manifests dir: %w", err)
	}

	chartTarPaths, err := getDownloadedCharts(helmChartPaths)
	if err != nil {
		return "", "", fmt.Errorf("getting downloaded helm chart paths: %w", err)
	}

	helmSrcDir := filepath.Join(ctx.ImageConfigDir, k8sDir, helmDir)
	err = writeUpdatedHelmManifests(k8sManifestsDestDir, chartTarPaths, helmManifestHolderDir, helmSrcDir)
	if err != nil {
		return "", "", fmt.Errorf("writing updated helm chart manifests: %w", err)
	}

	return helmTemplatePath, helmManifestHolderDir, nil
}
