package combustion

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	k8sComponentName  = "kubernetes"
	k8sConfigDir      = "kubernetes"
	k8sConfigFile     = "config.yaml"
	rke2InstallScript = "15-rke2-install.sh"
)

var (
	//go:embed templates/15-rke2-installer.sh.tpl
	rke2InstallerScript string
)

func configureKubernetes(ctx *image.Context) ([]string, error) {
	version := ctx.ImageDefinition.Kubernetes.Version

	if version == "" {
		log.AuditComponentSkipped(k8sComponentName)
		return nil, nil
	}

	// Show a message to the user to indicate that the Kubernetes component
	// is usually taking longer to complete due to downloading files
	log.Audit("Configuring Kubernetes component...")

	configureFunc := kubernetesConfigurator(version)
	if configureFunc == nil {
		log.AuditComponentFailed(k8sComponentName)
		return nil, fmt.Errorf("cannot configure kubernetes version: %s", version)
	}

	script, err := configureFunc(ctx)
	if err != nil {
		log.AuditComponentFailed(k8sComponentName)
		return nil, fmt.Errorf("configuring kubernetes components: %w", err)
	}

	log.AuditComponentSuccessful(k8sComponentName)
	return []string{script}, nil
}

func kubernetesConfigurator(version string) func(*image.Context) (string, error) {
	switch {
	case strings.Contains(version, image.KubernetesDistroRKE2):
		return configureRKE2
	case strings.Contains(version, image.KubernetesDistroK3S):
		return configureK3S
	default:
		return nil
	}
}

func installKubernetesScript(ctx *image.Context, distribution string) error {
	sourcePath := "/" // root level of the container image
	destPath := ctx.CombustionDir

	return ctx.KubernetesScriptInstaller.InstallScript(distribution, sourcePath, destPath)
}

func configureK3S(_ *image.Context) (string, error) {
	return "", fmt.Errorf("not implemented yet")
}

func configureRKE2(ctx *image.Context) (string, error) {
	if err := installKubernetesScript(ctx, image.KubernetesDistroRKE2); err != nil {
		return "", fmt.Errorf("copying RKE2 installer script: %w", err)
	}

	configFile, err := copyKubernetesConfig(ctx, image.KubernetesDistroRKE2)
	if err != nil {
		return "", fmt.Errorf("copying RKE2 config: %w", err)
	}

	installPath, imagesPath, err := ctx.KubernetesArtefactDownloader.DownloadArtefacts(
		ctx.ImageDefinition.Kubernetes,
		ctx.ImageDefinition.Image.Arch,
		ctx.CombustionDir,
	)
	if err != nil {
		return "", fmt.Errorf("downloading RKE2 artefacts: %w", err)
	}

	rke2 := struct {
		image.Kubernetes
		ConfigFile  string
		InstallPath string
		ImagesPath  string
	}{
		Kubernetes:  ctx.ImageDefinition.Kubernetes,
		ConfigFile:  configFile,
		InstallPath: installPath,
		ImagesPath:  imagesPath,
	}

	data, err := template.Parse(rke2InstallScript, rke2InstallerScript, &rke2)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 install template: %w", err)
	}

	installScript := filepath.Join(ctx.CombustionDir, rke2InstallScript)
	if err = os.WriteFile(installScript, []byte(data), fileio.ExecutablePerms); err != nil {
		return "", fmt.Errorf("writing RKE2 install script: %w", err)
	}

	return rke2InstallScript, nil
}

func copyKubernetesConfig(ctx *image.Context, distro string) (string, error) {
	if !isComponentConfigured(ctx, k8sConfigDir) {
		zap.S().Info("Kubernetes config file not provided")
		return "", nil
	}

	configDir := generateComponentPath(ctx, k8sConfigDir)
	configFile := filepath.Join(configDir, k8sConfigFile)

	_, err := os.Stat(configFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("kubernetes component directory exists but does not contain config.yaml")
		}
		return "", fmt.Errorf("error checking kubernetes config file: %w", err)
	}

	destFile := fmt.Sprintf("%s_config.yaml", distro)

	if err = fileio.CopyFile(configFile, filepath.Join(ctx.CombustionDir, destFile), fileio.NonExecutablePerms); err != nil {
		return "", fmt.Errorf("copying kubernetes config file: %w", err)
	}

	return destFile, nil
}
