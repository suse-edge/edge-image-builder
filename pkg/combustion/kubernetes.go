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
	"github.com/suse-edge/edge-image-builder/pkg/registry"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	k8sComponentName    = "kubernetes"
	k8sDir              = "kubernetes"
	k8sConfigDir        = "config"
	k8sServerConfigFile = "server.yaml"
	rke2InstallScript   = "15-rke2-install.sh"

	cniKey          = "cni"
	cniDefaultValue = image.CNITypeCilium
)

var (
	//go:embed templates/15-rke2-single-node-installer.sh.tpl
	rke2SingleNodeInstaller string
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

	manifestURLs := ctx.ImageDefinition.Kubernetes.Manifests.URLs
	localManifestsComponentDir := filepath.Join("kubernetes", "manifests")
	localManifestsConfigured := isComponentConfigured(ctx, localManifestsComponentDir)
	if localManifestsConfigured || len(manifestURLs) != 0 {
		localManifestsSrcDir := filepath.Join(ctx.ImageConfigDir, "kubernetes", "manifests")
		k8sCombustionDir := filepath.Join(ctx.CombustionDir, k8sDir)
		err = os.MkdirAll(k8sCombustionDir, os.ModePerm)
		if err != nil {
			log.AuditComponentFailed(k8sComponentName)
			return nil, fmt.Errorf("creating kubernetes combustion dir: %w", err)
		}

		err = configureManifests(k8sCombustionDir, localManifestsConfigured, localManifestsSrcDir, manifestURLs)
		if err != nil {
			log.AuditComponentFailed(k8sComponentName)
			return nil, fmt.Errorf("configuring kubernetes manifests: %w", err)
		}
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

	config, err := parseKubernetesConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 config: %w", err)
	}

	cni, multusEnabled, err := extractCNI(config)
	if err != nil {
		return "", fmt.Errorf("extracting CNI from RKE2 config: %w", err)
	}

	configFile, err := storeKubernetesConfig(ctx, config, image.KubernetesDistroRKE2)
	if err != nil {
		return "", fmt.Errorf("storing RKE2 config file: %w", err)
	}

	installPath, imagesPath, err := ctx.KubernetesArtefactDownloader.DownloadArtefacts(
		ctx.ImageDefinition.Image.Arch,
		ctx.ImageDefinition.Kubernetes.Version,
		cni,
		multusEnabled,
		ctx.CombustionDir,
	)
	if err != nil {
		return "", fmt.Errorf("downloading RKE2 artefacts: %w", err)
	}

	manifestsPath := ""
	if isComponentConfigured(ctx, filepath.Join(k8sDir, "manifests")) {
		manifestsPath = filepath.Join(k8sDir, "manifests")
	}

	rke2 := struct {
		image.Kubernetes
		ConfigFile    string
		InstallPath   string
		ImagesPath    string
		ManifestsPath string
	}{
		Kubernetes:    ctx.ImageDefinition.Kubernetes,
		ConfigFile:    configFile,
		InstallPath:   installPath,
		ImagesPath:    imagesPath,
		ManifestsPath: manifestsPath,
	}

	data, err := template.Parse(rke2InstallScript, rke2SingleNodeInstaller, &rke2)
	if err != nil {
		return "", fmt.Errorf("parsing RKE2 install template: %w", err)
	}

	installScript := filepath.Join(ctx.CombustionDir, rke2InstallScript)
	if err = os.WriteFile(installScript, []byte(data), fileio.ExecutablePerms); err != nil {
		return "", fmt.Errorf("writing RKE2 install script: %w", err)
	}

	return rke2InstallScript, nil
}

func parseKubernetesConfig(ctx *image.Context) (map[string]any, error) {
	auditDefaultCNI := func() {
		auditMessage := fmt.Sprintf("Kubernetes CNI not explicitly set, defaulting to: %s", cniDefaultValue)
		log.Audit(auditMessage)
	}

	config := map[string]any{}

	configDir := generateComponentPath(ctx, k8sDir)
	configFile := filepath.Join(configDir, k8sConfigDir, k8sServerConfigFile)

	b, err := os.ReadFile(configFile)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("reading kubernetes config file: %w", err)
		}

		auditDefaultCNI()
		zap.S().Infof("Kubernetes server config file not provided, proceeding with CNI: %s", cniDefaultValue)

		config[cniKey] = cniDefaultValue
		return config, nil
	}

	if err = yaml.Unmarshal(b, &config); err != nil {
		return nil, fmt.Errorf("parsing kubernetes config file: %w", err)
	}

	if _, ok := config[cniKey]; !ok {
		auditDefaultCNI()
		zap.S().Infof("CNI not set in config file, proceeding with CNI: %s", cniDefaultValue)

		config[cniKey] = cniDefaultValue
	}

	return config, nil
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

func storeKubernetesConfig(ctx *image.Context, config map[string]any, distribution string) (string, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("serializing kubernetes config: %w", err)
	}

	configFile := fmt.Sprintf("%s_config.yaml", distribution)
	configPath := filepath.Join(ctx.CombustionDir, configFile)

	if err = os.WriteFile(configPath, data, fileio.NonExecutablePerms); err != nil {
		return "", fmt.Errorf("storing kubernetes config file: %w", err)
	}

	return configFile, nil
}

func configureManifests(k8sCombustionDir string, localManifestsConfigured bool, localManifestsSrcDir string, manifestURLs []string) error {
	manifestDestDir := filepath.Join(k8sCombustionDir, "manifests")
	err := os.Mkdir(manifestDestDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating manifests destination dir: %w", err)
	}

	if localManifestsConfigured {
		_, err = registry.CopyManifests(localManifestsSrcDir, manifestDestDir)
		if err != nil {
			return fmt.Errorf("copying local manifests to combustion dir: %w", err)
		}
	}

	if len(manifestURLs) != 0 {
		_, err = registry.DownloadManifests(manifestURLs, manifestDestDir)
		if err != nil {
			return fmt.Errorf("downloading manifests to combustion dir: %w", err)
		}
	}

	return nil
}
