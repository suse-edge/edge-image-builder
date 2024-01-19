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
	"gopkg.in/yaml.v3"
)

const (
	k8sComponentName    = "kubernetes"
	k8sDir              = "kubernetes"
	k8sConfigDir        = "config"
	k8sServerConfigFile = "server.yaml"
	k8sAgentConfigFile  = "agent.yaml"
	rke2InstallScript   = "15-rke2-install.sh"

	tokenKey        = "token"
	cniKey          = "cni"
	cniDefaultValue = image.CNITypeCilium
	serverKey       = "server"
	tlsSANKey       = "tls-san"
)

var (
	//go:embed templates/15-rke2-single-node-installer.sh.tpl
	rke2SingleNodeInstaller string

	//go:embed templates/15-rke2-multi-node-installer.sh.tpl
	rke2MultiNodeInstaller string

	//go:embed templates/rke2-ha-api.yaml.tpl
	rke2HAAPIManifest string
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

	if len(ctx.ImageDefinition.Kubernetes.Nodes) > 1 {
		zap.S().Info("Configuring multi node RKE2 cluster...")
		return configureMultiNodeRKE2(ctx)
	}

	zap.S().Info("Configuring single node RKE2 cluster...")
	return configureSingleNodeRKE2(ctx)
}

func configureSingleNodeRKE2(ctx *image.Context) (string, error) {
	serverConfig, err := parseKubernetesConfig(ctx, k8sServerConfigFile)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 server config: %w", err)
	}

	// Establish sane default values
	setClusterCNI(serverConfig)
	delete(serverConfig, tlsSANKey) // TODO: Figure out whether tls-san is fine to keep
	delete(serverConfig, serverKey)

	if err = storeKubernetesConfig(ctx, serverConfig, k8sServerConfigFile); err != nil {
		return "", fmt.Errorf("storing RKE2 server config file: %w", err)
	}

	installPath, imagesPath, err := downloadRKE2Artefacts(ctx, serverConfig)
	if err != nil {
		return "", fmt.Errorf("downloading RKE2 artefacts: %w", err)
	}

	rke2 := struct {
		ConfigFile  string
		InstallPath string
		ImagesPath  string
	}{
		ConfigFile:  k8sServerConfigFile,
		InstallPath: installPath,
		ImagesPath:  imagesPath,
	}

	return storeRKE2Installer(ctx, "single-node-rke2", rke2SingleNodeInstaller, &rke2)
}

func configureMultiNodeRKE2(ctx *image.Context) (string, error) {
	firstNode := findKubernetesInitialiserNode(&ctx.ImageDefinition.Kubernetes)
	if firstNode == "" {
		return "", fmt.Errorf("failed to determine first node in cluster")
	}

	serverConfig, err := parseKubernetesConfig(ctx, k8sServerConfigFile)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 server config: %w", err)
	}

	// Establish sane default values
	setClusterCNI(serverConfig)
	setClusterToken(serverConfig)
	setClusterAPIHost(serverConfig, ctx.ImageDefinition.Kubernetes.Network.APIHost)
	setClusterAPIAddress(serverConfig, ctx.ImageDefinition.Kubernetes.Network.APIVIP)

	if err = storeKubernetesConfig(ctx, serverConfig, k8sServerConfigFile); err != nil {
		return "", fmt.Errorf("storing RKE2 server config file: %w", err)
	}

	agentConfig, err := parseKubernetesConfig(ctx, k8sAgentConfigFile)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 agent config: %w", err)
	}

	// Ensure the agent uses the same cluster configuration values as the server
	agentConfig[tokenKey] = serverConfig[tokenKey]
	agentConfig[cniKey] = serverConfig[cniKey]
	agentConfig[serverKey] = serverConfig[serverKey]
	agentConfig[tlsSANKey] = serverConfig[tlsSANKey]

	if err = storeKubernetesConfig(ctx, agentConfig, k8sAgentConfigFile); err != nil {
		return "", fmt.Errorf("storing RKE2 agent config file: %w", err)
	}

	// Drop values not applicable to the initialiser
	delete(serverConfig, tlsSANKey)
	delete(serverConfig, serverKey)

	serverConfigFile := fmt.Sprintf("first_%s", k8sServerConfigFile)

	if err = storeKubernetesConfig(ctx, serverConfig, serverConfigFile); err != nil {
		return "", fmt.Errorf("storing RKE2 initialising server config file: %w", err)
	}

	installPath, imagesPath, err := downloadRKE2Artefacts(ctx, serverConfig)
	if err != nil {
		return "", fmt.Errorf("downloading RKE2 artefacts: %w", err)
	}

	haManifest, err := storeHighAvailabilityRKE2Manifest(ctx)
	if err != nil {
		return "", fmt.Errorf("storing RKE2 HA API manifest: %w", err)
	}

	rke2 := struct {
		image.Kubernetes
		FirstNode   string
		HAManifest  string
		InstallPath string
		ImagesPath  string
	}{
		Kubernetes:  ctx.ImageDefinition.Kubernetes,
		FirstNode:   firstNode,
		HAManifest:  haManifest,
		InstallPath: installPath,
		ImagesPath:  imagesPath,
	}

	return storeRKE2Installer(ctx, "multi-node-rke2", rke2MultiNodeInstaller, &rke2)
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

func downloadRKE2Artefacts(ctx *image.Context, clusterConfig map[string]any) (installPath, imagesPath string, err error) {
	cni, multusEnabled, err := extractCNI(clusterConfig)
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

func findKubernetesInitialiserNode(kubernetes *image.Kubernetes) string {
	for _, node := range kubernetes.Nodes {
		if node.First {
			return node.Hostname
		}
	}

	// Use the first server node as an initialiser
	for _, node := range kubernetes.Nodes {
		if node.Type == image.KubernetesNodeTypeServer {
			zap.S().Infof("Using '%s' for cluster initialiser since none of the nodes was explicitly configured", node.Hostname)
			return node.Hostname
		}
	}

	return ""
}

func storeHighAvailabilityRKE2Manifest(ctx *image.Context) (string, error) {
	const haManifest = "rke2-ha-api.yaml"

	manifest := struct {
		APIAddress string
	}{
		APIAddress: ctx.ImageDefinition.Kubernetes.Network.APIVIP,
	}

	data, err := template.Parse("rke2-ha-api", rke2HAAPIManifest, &manifest)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 HA API template: %w", err)
	}

	installScript := filepath.Join(ctx.CombustionDir, haManifest)
	if err = os.WriteFile(installScript, []byte(data), fileio.ExecutablePerms); err != nil {
		return "", fmt.Errorf("writing RKE2 HA API manifest: %w", err)
	}

	return haManifest, nil
}

func parseKubernetesConfig(ctx *image.Context, configFile string) (map[string]any, error) {
	config := map[string]any{}

	configDir := generateComponentPath(ctx, k8sDir)
	file := filepath.Join(configDir, k8sConfigDir, configFile)

	b, err := os.ReadFile(file)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("reading kubernetes config file '%s': %w", configFile, err)
		}

		// Use an empty config which will be automatically populated later
		return config, nil
	}

	if err = yaml.Unmarshal(b, &config); err != nil {
		return nil, fmt.Errorf("parsing kubernetes config file '%s': %w", configFile, err)
	}

	return config, nil
}

func setClusterToken(config map[string]any) {
	if _, ok := config[tokenKey].(string); ok {
		return
	}

	token := "foobar" // TODO: generate

	zap.S().Infof("Generated cluster token: %s", token)
	config[tokenKey] = token
}

func setClusterCNI(config map[string]any) {
	if _, ok := config[cniKey]; ok {
		return
	}

	auditMessage := fmt.Sprintf("Kubernetes CNI not explicitly set, defaulting to: %s", cniDefaultValue)
	log.Audit(auditMessage)

	zap.S().Infof("CNI not set in config file, proceeding with CNI: %s", cniDefaultValue)

	config[cniKey] = cniDefaultValue
}

func setClusterAPIHost(config map[string]any, apiHost string) {
	if apiHost == "" {
		zap.S().Warnf("Attempted to set an empty cluster API host")
		return
	}

	tlsSAN, ok := config[tlsSANKey]
	if !ok {
		config[tlsSANKey] = []string{apiHost}
		return
	}

	switch v := tlsSAN.(type) {
	case string:
		config[tlsSANKey] = []string{v, apiHost}
	case []string:
		v = append(v, apiHost)
		config[tlsSANKey] = v
	case []any:
		v = append(v, apiHost)
		config[tlsSANKey] = v
	default:
		zap.S().Warnf("Ignoring invalid 'tls-san' value: %s", v)
		config[tlsSANKey] = []string{apiHost}
	}
}

func setClusterAPIAddress(config map[string]any, apiAddress string) {
	if apiAddress == "" {
		zap.S().Warnf("Attempted to set an empty cluster API address")
		return
	}

	config[serverKey] = fmt.Sprintf("https://%s:9345", apiAddress)
}

func extractCNI(config map[string]any) (cni string, multusEnabled bool, err error) {
	switch configuredCNI := config[cniKey].(type) {
	case string:
		if configuredCNI == "" {
			return "", false, fmt.Errorf("cni not configured")
		}

		var cnis []string
		for _, cni = range strings.Split(configuredCNI, ",") {
			cnis = append(cnis, strings.TrimSpace(cni))
		}

		return parseCNIs(cnis)

	case []string:
		return parseCNIs(configuredCNI)

	case []any:
		var cnis []string
		for _, cni := range configuredCNI {
			c, ok := cni.(string)
			if !ok {
				return "", false, fmt.Errorf("invalid cni value: %v", cni)
			}
			cnis = append(cnis, c)
		}

		return parseCNIs(cnis)

	default:
		return "", false, fmt.Errorf("invalid cni: %v", configuredCNI)
	}
}

func parseCNIs(cnis []string) (cni string, multusEnabled bool, err error) {
	const multusPlugin = "multus"

	switch len(cnis) {
	case 1:
		cni = cnis[0]
		if cni == multusPlugin {
			return "", false, fmt.Errorf("multus must be used alongside another primary cni selection")
		}
	case 2:
		if cnis[0] == multusPlugin {
			cni = cnis[1]
			multusEnabled = true
		} else {
			return "", false, fmt.Errorf("multiple cni values are only allowed if multus is the first one")
		}
	default:
		return "", false, fmt.Errorf("invalid cni value: %v", cnis)
	}

	return cni, multusEnabled, nil
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
