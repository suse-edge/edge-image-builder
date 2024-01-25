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
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	k8sComponentName    = "kubernetes"
	k8sDir              = "kubernetes"
	k8sConfigDir        = "config"
	k8sServerConfigFile = "server.yaml"
	k8sAgentConfigFile  = "agent.yaml"
	rke2InstallScript   = "15-rke2-install.sh"
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

	if err = storeKubernetesConfig(ctx, cluster.ServerConfig, k8sServerConfigFile); err != nil {
		return "", fmt.Errorf("storing RKE2 server config file: %w", err)
	}

	installPath, imagesPath, err := downloadRKE2Artefacts(ctx, cluster)
	if err != nil {
		return "", fmt.Errorf("downloading RKE2 artefacts: %w", err)
	}

	templateValues := map[string]any{
		"apiVIP":      ctx.ImageDefinition.Kubernetes.Network.APIVIP,
		"apiHost":     ctx.ImageDefinition.Kubernetes.Network.APIHost,
		"installPath": installPath,
		"imagesPath":  imagesPath,
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

	if err = storeKubernetesConfig(ctx, cluster.AgentConfig, k8sAgentConfigFile); err != nil {
		return "", fmt.Errorf("storing RKE2 agent config file: %w", err)
	}

	initialiserConfigFile := fmt.Sprintf("init_%s", k8sServerConfigFile)
	if err = storeKubernetesConfig(ctx, cluster.InitialiserConfig, initialiserConfigFile); err != nil {
		return "", fmt.Errorf("storing RKE2 initialising server config file: %w", err)
	}

	vipManifest, err := storeRKE2VIPManifest(ctx)
	if err != nil {
		return "", fmt.Errorf("storing RKE2 VIP manifest: %w", err)
	}

	templateValues["nodes"] = ctx.ImageDefinition.Kubernetes.Nodes
	templateValues["initialiser"] = cluster.Initialiser
	templateValues["initialiserConfigFile"] = initialiserConfigFile
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

func storeKubernetesConfig(ctx *image.Context, config map[string]any, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("serializing kubernetes config: %w", err)
	}

	configPath := filepath.Join(ctx.CombustionDir, filename)

	if err = os.WriteFile(configPath, data, fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("storing kubernetes config file: %w", err)
	}

	return nil
}
