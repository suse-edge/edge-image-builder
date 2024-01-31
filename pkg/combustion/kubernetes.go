package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/kubernetes"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/registry"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	k8sComponentName        = "kubernetes"
	k8sDir                  = "kubernetes"
	k8sConfigDir            = "config"
	manifestsDir            = "manifests"
	k8sInitServerConfigFile = "init_server.yaml"
	k8sServerConfigFile     = "server.yaml"
	k8sAgentConfigFile      = "agent.yaml"
	rke2InstallScript       = "15-rke2-install.sh"
)

var (
	//go:embed templates/15-rke2-single-node-installer.sh.tpl
	rke2SingleNodeInstaller string

	//go:embed templates/15-rke2-multi-node-installer.sh.tpl
	rke2MultiNodeInstaller string

	//go:embed templates/rke2-vip.yaml.tpl
	rke2VIPManifest string
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

	if kubernetes.ServersCount(ctx.ImageDefinition.Kubernetes.Nodes) == 2 {
		log.Audit("WARNING: Kubernetes clusters consisting of two server nodes cannot form a highly available architecture")
		zap.S().Warn("Kubernetes cluster of two server nodes has been requested")
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
	zap.S().Info("Configuring RKE2 cluster")

	if err := installKubernetesScript(ctx, image.KubernetesDistroRKE2); err != nil {
		return "", fmt.Errorf("copying RKE2 installer script: %w", err)
	}

	configDir := generateComponentPath(ctx, k8sDir)
	configPath := filepath.Join(configDir, k8sConfigDir)

	cluster, err := kubernetes.NewCluster(&ctx.ImageDefinition.Kubernetes, configPath)
	if err != nil {
		return "", fmt.Errorf("initialising kubernetes cluster config: %w", err)
	}

	if err = storeKubernetesClusterConfig(cluster, ctx.CombustionDir); err != nil {
		return "", fmt.Errorf("storing RKE2 cluster config: %w", err)
	}

	installPath, imagesPath, err := downloadRKE2Artefacts(ctx, cluster)
	if err != nil {
		return "", fmt.Errorf("downloading RKE2 artefacts: %w", err)
	}

	manifestsPath, err := configureManifests(ctx)
	if err != nil {
		return "", fmt.Errorf("configuring kubernetes manifests: %w", err)
	}

	templateValues := map[string]any{
		"apiVIP":          ctx.ImageDefinition.Kubernetes.Network.APIVIP,
		"apiHost":         ctx.ImageDefinition.Kubernetes.Network.APIHost,
		"installPath":     installPath,
		"imagesPath":      imagesPath,
		"manifestsPath":   manifestsPath,
		"registryMirrors": registryMirrorsFileName,
	}

	singleNode := len(ctx.ImageDefinition.Kubernetes.Nodes) < 2
	if singleNode {
		var vipManifest string

		if ctx.ImageDefinition.Kubernetes.Network.APIVIP == "" {
			zap.S().Info("Virtual IP address for RKE2 cluster is not provided and will not be configured")
		} else if vipManifest, err = storeRKE2VIPManifest(ctx); err != nil {
			return "", fmt.Errorf("storing RKE2 VIP manifest: %w", err)
		}

		templateValues["configFile"] = k8sServerConfigFile
		templateValues["vipManifest"] = vipManifest

		return storeRKE2Installer(ctx, "single-node-rke2", rke2SingleNodeInstaller, templateValues)
	}

	vipManifest, err := storeRKE2VIPManifest(ctx)
	if err != nil {
		return "", fmt.Errorf("storing RKE2 VIP manifest: %w", err)
	}

	templateValues["nodes"] = ctx.ImageDefinition.Kubernetes.Nodes
	templateValues["initialiser"] = cluster.InitialiserName
	templateValues["initialiserConfigFile"] = k8sInitServerConfigFile
	templateValues["vipManifest"] = vipManifest

	return storeRKE2Installer(ctx, "multi-node-rke2", rke2MultiNodeInstaller, templateValues)
}

func storeRKE2Installer(ctx *image.Context, templateName, templateContents string, templateValues any) (string, error) {
	data, err := template.Parse(templateName, templateContents, templateValues)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 install template: %w", err)
	}

	installScript := filepath.Join(ctx.CombustionDir, rke2InstallScript)
	if err = os.WriteFile(installScript, []byte(data), fileio.ExecutablePerms); err != nil {
		return "", fmt.Errorf("writing RKE2 install script: %w", err)
	}

	return rke2InstallScript, nil
}

func downloadRKE2Artefacts(ctx *image.Context, cluster *kubernetes.Cluster) (installPath, imagesPath string, err error) {
	cni, multusEnabled, err := cluster.ExtractCNI()
	if err != nil {
		return "", "", fmt.Errorf("extracting CNI from cluster config: %w", err)
	}

	return ctx.KubernetesArtefactDownloader.DownloadArtefacts(
		ctx.ImageDefinition.Image.Arch,
		ctx.ImageDefinition.Kubernetes.Version,
		cni,
		multusEnabled,
		ctx.CombustionDir,
	)
}

func storeRKE2VIPManifest(ctx *image.Context) (string, error) {
	const vipManifest = "rke2-vip.yaml"

	manifest := struct {
		APIAddress string
	}{
		APIAddress: ctx.ImageDefinition.Kubernetes.Network.APIVIP,
	}

	data, err := template.Parse("rke2-vip", rke2VIPManifest, &manifest)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 VIP template: %w", err)
	}

	installScript := filepath.Join(ctx.CombustionDir, vipManifest)
	if err = os.WriteFile(installScript, []byte(data), fileio.NonExecutablePerms); err != nil {
		return "", fmt.Errorf("writing RKE2 VIP manifest: %w", err)
	}

	return vipManifest, nil
}

func storeKubernetesClusterConfig(cluster *kubernetes.Cluster, destPath string) error {
	serverConfig := filepath.Join(destPath, k8sServerConfigFile)
	if err := storeKubernetesConfig(cluster.ServerConfig, serverConfig); err != nil {
		return fmt.Errorf("storing server config file: %w", err)
	}

	if cluster.InitialiserConfig != nil {
		initialiserConfig := filepath.Join(destPath, k8sInitServerConfigFile)

		if err := storeKubernetesConfig(cluster.InitialiserConfig, initialiserConfig); err != nil {
			return fmt.Errorf("storing init server config file: %w", err)
		}
	}

	if cluster.AgentConfig != nil {
		agentConfig := filepath.Join(destPath, k8sAgentConfigFile)

		if err := storeKubernetesConfig(cluster.AgentConfig, agentConfig); err != nil {
			return fmt.Errorf("storing agent config file: %w", err)
		}
	}

	return nil
}

func storeKubernetesConfig(config map[string]any, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("serializing kubernetes config: %w", err)
	}

	return os.WriteFile(configPath, data, fileio.NonExecutablePerms)
}

func configureManifests(ctx *image.Context) (string, error) {
	manifestURLs := ctx.ImageDefinition.Kubernetes.Manifests.URLs
	localManifestsConfigured := isComponentConfigured(ctx, filepath.Join(k8sDir, manifestsDir))

	if !localManifestsConfigured && len(manifestURLs) == 0 {
		return "", nil
	}

	manifestsPath := filepath.Join(k8sDir, manifestsDir)
	manifestDestDir := filepath.Join(ctx.CombustionDir, manifestsPath)
	err := os.Mkdir(manifestDestDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("creating manifests destination dir: %w", err)
	}

	if localManifestsConfigured {
		localManifestsSrcDir := filepath.Join(ctx.ImageConfigDir, k8sDir, manifestsDir)
		err = fileio.CopyFiles(localManifestsSrcDir, manifestDestDir, ".yaml", false)
		if err != nil {
			return "", fmt.Errorf("copying local manifests to combustion dir: %w", err)
		}
		err = fileio.CopyFiles(localManifestsSrcDir, manifestDestDir, ".yml", false)
		if err != nil {
			return "", fmt.Errorf("copying local manifests to combustion dir: %w", err)
		}
	}

	if len(manifestURLs) != 0 {
		_, err = registry.DownloadManifests(manifestURLs, manifestDestDir)
		if err != nil {
			return "", fmt.Errorf("downloading manifests to combustion dir: %w", err)
		}
	}

	return manifestsPath, nil
}
