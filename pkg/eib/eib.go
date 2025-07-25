package eib

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/rpm/resolver"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/cache"
	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/container"
	"github.com/suse-edge/edge-image-builder/pkg/helm"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/kubernetes"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/network"
	"github.com/suse-edge/edge-image-builder/pkg/registry"
	"go.uber.org/zap"
)

func Run(ctx *image.Context, rootBuildDir string) error {
	if err := appendKubernetesSELinuxRPMs(ctx); err != nil {
		log.Auditf("Bootstrapping dependency services failed.")
		return fmt.Errorf("configuring kubernetes selinux policy: %w", err)
	}

	appendElementalRPMs(ctx)
	appendFIPS(ctx)
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

	selinuxPackage, err := kubernetes.SELinuxPackage(ctx.ImageDefinition.Kubernetes.Version, ctx.ArtifactSources)
	if err != nil {
		return fmt.Errorf("identifying selinux package: %w", err)
	}

	repository, err := kubernetes.SELinuxRepository(ctx.ImageDefinition.Kubernetes.Version, ctx.ArtifactSources)
	if err != nil {
		return fmt.Errorf("identifying selinux repository: %w", err)
	}

	appendRPMs(ctx, []image.AddRepo{repository}, selinuxPackage)

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

	rpmsPath := combustion.RPMsPath(ctx)
	rpmDirEntries, err := os.ReadDir(rpmsPath)
	if err != nil && !os.IsNotExist(err) {
		zap.S().Warnf("Looking for '%s' dir failed unexpectedly: %s", rpmsPath, err)
	}

	if !slices.ContainsFunc(rpmDirEntries, func(entry os.DirEntry) bool {
		return strings.Contains(entry.Name(), combustion.ElementalPackages[0])
	}) {
		log.AuditInfo("Elemental registration is configured. The necessary RPM packages will be downloaded.")
		appendRPMs(ctx, nil, combustion.ElementalPackages...)
	}
}

func appendFIPS(ctx *image.Context) {
	fips := ctx.ImageDefinition.OperatingSystem.EnableFIPS
	if fips {
		log.AuditInfo("FIPS mode is configured. The necessary RPM packages will be downloaded.")

		packages := ctx.ImageDefinition.OperatingSystem.Packages
		if packages.RegCode == "" && len(packages.AdditionalRepos) > 0 {
			log.Audit("WARNING: FIPS enabled with no SUSE registration code provided. Package resolution may fail if additional repositories do not contain the `patterns-base-fips` package.")
			zap.S().Warn("Detected FIPS for installation with no sccRegistrationCode provided")
		}

		appendRPMs(ctx, nil, combustion.FIPSPackages...)
		appendKernelArgs(ctx, combustion.FIPSKernelArgs...)
	}
}

func appendRPMs(ctx *image.Context, repos []image.AddRepo, packages ...string) {
	repositories := ctx.ImageDefinition.OperatingSystem.Packages.AdditionalRepos
	repositories = append(repositories, repos...)

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

func appendKernelArgs(ctx *image.Context, kernelArgs ...string) {
	kernelArgList := ctx.ImageDefinition.OperatingSystem.KernelArgs
	kernelArgList = append(kernelArgList, kernelArgs...)
	ctx.ImageDefinition.OperatingSystem.KernelArgs = kernelArgList
}

func buildCombustion(ctx *image.Context, rootDir string) (*combustion.Combustion, error) {
	cacheDir := filepath.Join(rootDir, "cache")
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating a cache directory: %w", err)
	}
	ctx.CacheDir = cacheDir

	combustionHandler := &combustion.Combustion{
		NetworkConfigGenerator:       network.ConfigGenerator{},
		NetworkConfiguratorInstaller: network.ConfiguratorInstaller{},
	}

	if !combustion.SkipRPMComponent(ctx) || combustion.IsEmbeddedArtifactRegistryConfigured(ctx) {
		p, err := podman.New(ctx.BuildDir)
		if err != nil {
			return nil, fmt.Errorf("setting up Podman instance: %w", err)
		}

		combustionHandler.ImageDigester = &container.ImageDigester{
			ImageInspector: p,
		}

		if !combustion.SkipRPMComponent(ctx) {
			imgPath := filepath.Join(ctx.ImageConfigDir, "base-images", ctx.ImageDefinition.Image.BaseImage)
			imgType := ctx.ImageDefinition.Image.ImageType
			luksKey := ctx.ImageDefinition.OperatingSystem.RawConfiguration.LUKSKey
			baseBuilder := resolver.NewTarballBuilder(ctx.BuildDir, imgPath, imgType, string(ctx.ImageDefinition.Image.Arch), luksKey, p)

			combustionHandler.RPMResolver = resolver.New(ctx.BuildDir, p, baseBuilder, "", string(ctx.ImageDefinition.Image.Arch))
			combustionHandler.RPMRepoCreator = rpm.NewRepoCreator(ctx.BuildDir)
		}

		if combustion.IsEmbeddedArtifactRegistryConfigured(ctx) {
			helmClient := helm.New(ctx.BuildDir, combustion.HelmCertsPath(ctx))

			combustionHandler.Registry, err = registry.New(ctx, combustion.KubernetesManifestsPath(ctx), helmClient, combustion.HelmValuesPath(ctx))
			if err != nil {
				return nil, fmt.Errorf("initialising embedded artifact registry: %w", err)
			}
		}
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		c, err := cache.New(cacheDir)
		if err != nil {
			return nil, fmt.Errorf("initialising cache instance: %w", err)
		}

		combustionHandler.KubernetesScriptDownloader = kubernetes.ScriptDownloader{}
		combustionHandler.KubernetesArtefactDownloader = kubernetes.ArtefactDownloader{
			Cache:          c,
			Rke2ReleaseURL: ctx.ArtifactSources.Kubernetes.Rke2.ReleaseURL,
			K3sReleaseURL:  ctx.ArtifactSources.Kubernetes.K3s.ReleaseURL,
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
