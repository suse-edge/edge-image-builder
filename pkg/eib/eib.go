package eib

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/cache"
	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/env"
	"github.com/suse-edge/edge-image-builder/pkg/helm"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/kubernetes"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/network"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/rpm/resolver"
	"go.uber.org/zap"
)

func Run(ctx *image.Context, rootBuildDir string) error {
	if err := appendKubernetesSELinuxRPMs(ctx); err != nil {
		log.Auditf("Bootstrapping dependency services failed.")
		return fmt.Errorf("configuring kubernetes selinux policy: %w", err)
	}

	appendElementalRPMs(ctx)
	appendHelm(ctx)

	c, err := buildCombustion(ctx, rootBuildDir)
	if err != nil {
		log.Audit("Bootstrapping dependency services failed.")
		return fmt.Errorf("building combustion: %w", err)
	}

	builder := build.NewBuilder(ctx, c)
	return builder.Build()
}

func appendKubernetesSELinuxRPMs(ctx *image.Context) error {
	if ctx.ImageDefinition.Kubernetes.Version == "" {
		return nil
	}

	configPath := combustion.KubernetesConfigPath(ctx)
	config, err := kubernetes.ParseKubernetesConfig(configPath)
	if err != nil {
		return fmt.Errorf("parsing kubernetes server config: %w", err)
	}

	selinuxEnabled, _ := config["selinux"].(bool)
	if !selinuxEnabled {
		return nil
	}

	log.AuditInfo("SELinux is enabled in the Kubernetes configuration. " +
		"The necessary RPM packages will be downloaded.")

	selinuxPackage, err := kubernetes.SELinuxPackage(ctx.ImageDefinition.Kubernetes.Version)
	if err != nil {
		return fmt.Errorf("identifying selinux package: %w", err)
	}

	repository, err := kubernetes.SELinuxRepository(ctx.ImageDefinition.Kubernetes.Version)
	if err != nil {
		return fmt.Errorf("identifying selinux repository: %w", err)
	}

	appendRPMs(ctx, repository, selinuxPackage)

	gpgKeysDir := combustion.GPGKeysPath(ctx)
	if err = os.MkdirAll(gpgKeysDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating directory '%s': %w", gpgKeysDir, err)
	}

	if err = kubernetes.DownloadSELinuxRPMsSigningKey(gpgKeysDir); err != nil {
		return fmt.Errorf("downloading signing key: %w", err)
	}

	return nil
}

func appendElementalRPMs(ctx *image.Context) {
	elementalDir := combustion.ElementalPath(ctx)
	if _, err := os.Stat(elementalDir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.S().Warnf("Looking for '%s' dir failed unexpectedly: %s", elementalDir, err)
		}

		return
	}

	log.AuditInfo("Elemental registration is configured. The necessary RPM packages will be downloaded.")

	appendRPMs(ctx, image.AddRepo{URL: env.ElementalPackageRepository}, combustion.ElementalPackages...)
}

func appendRPMs(ctx *image.Context, repository image.AddRepo, packages ...string) {
	repositories := ctx.ImageDefinition.OperatingSystem.Packages.AdditionalRepos
	repositories = append(repositories, repository)

	packageList := ctx.ImageDefinition.OperatingSystem.Packages.PKGList
	packageList = append(packageList, packages...)

	ctx.ImageDefinition.OperatingSystem.Packages.PKGList = packageList
	ctx.ImageDefinition.OperatingSystem.Packages.AdditionalRepos = repositories
}

func appendHelm(ctx *image.Context) {
	componentCharts, componentRepos := combustion.ComponentHelmCharts(ctx)

	ctx.ImageDefinition.Kubernetes.Helm.Charts = append(ctx.ImageDefinition.Kubernetes.Helm.Charts, componentCharts...)
	ctx.ImageDefinition.Kubernetes.Helm.Repositories = append(ctx.ImageDefinition.Kubernetes.Helm.Repositories, componentRepos...)
}

func buildCombustion(ctx *image.Context, rootDir string) (*combustion.Combustion, error) {
	combustionHandler := &combustion.Combustion{
		NetworkConfigGenerator:       network.ConfigGenerator{},
		NetworkConfiguratorInstaller: network.ConfiguratorInstaller{},
	}

	if !combustion.SkipRPMComponent(ctx) {
		p, err := podman.New(ctx.BuildDir)
		if err != nil {
			return nil, fmt.Errorf("setting up Podman instance: %w", err)
		}

		imgPath := filepath.Join(ctx.ImageConfigDir, "base-images", ctx.ImageDefinition.Image.BaseImage)
		imgType := ctx.ImageDefinition.Image.ImageType
		baseBuilder := resolver.NewTarballBuilder(ctx.BuildDir, imgPath, imgType, p)

		combustionHandler.RPMResolver = resolver.New(ctx.BuildDir, p, baseBuilder, "")
		combustionHandler.RPMRepoCreator = rpm.NewRepoCreator(ctx.BuildDir)
	}

	if combustion.IsEmbeddedArtifactRegistryConfigured(ctx) {
		certsDir := filepath.Join(ctx.ImageConfigDir, combustion.K8sDir, combustion.HelmDir, combustion.CertsDir)
		combustionHandler.HelmClient = helm.New(ctx.BuildDir, certsDir)
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		c, err := cache.New(rootDir)
		if err != nil {
			return nil, fmt.Errorf("initialising cache instance: %w", err)
		}

		combustionHandler.KubernetesScriptDownloader = kubernetes.ScriptDownloader{}
		combustionHandler.KubernetesArtefactDownloader = kubernetes.ArtefactDownloader{
			Cache: c,
		}
	}

	return combustionHandler, nil
}

func SetupBuildDirectory(rootDir string) (string, error) {
	timestamp := time.Now().Format("Jan02_15-04-05")
	buildDir := filepath.Join(rootDir, fmt.Sprintf("build-%s", timestamp))
	if err := os.MkdirAll(buildDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("creating a build directory: %w", err)
	}

	return buildDir, nil
}

func SetupCombustionDirectory(buildDir string) (combustionDir, artefactsDir string, err error) {
	combustionDir = filepath.Join(buildDir, "combustion")
	if err = os.MkdirAll(combustionDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating a combustion directory: %w", err)
	}

	artefactsDir = filepath.Join(buildDir, "artefacts")
	if err = os.MkdirAll(artefactsDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating an artefacts directory: %w", err)
	}

	return combustionDir, artefactsDir, nil
}
