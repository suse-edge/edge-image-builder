package build

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/cache"
	"github.com/suse-edge/edge-image-builder/pkg/cli/cmd"
	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/helm"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/kubernetes"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/network"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/rpm/resolver"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

const (
	buildLogFilename     = "eib-build.log"
	checkBuildLogMessage = "Please check the eib-build.log file under the build directory for more information."
)

func Run(_ *cli.Context) error {
	args := &cmd.BuildArgs

	rootBuildDir := args.RootBuildDir
	if rootBuildDir == "" {
		const defaultBuildDir = "_build"

		rootBuildDir = filepath.Join(args.ConfigDir, defaultBuildDir)
		if err := os.MkdirAll(rootBuildDir, os.ModePerm); err != nil {
			log.Auditf("The root build directory could not be set up under the configuration directory '%s'.", args.ConfigDir)
			return err
		}
	}

	buildDir, combustionDir, err := build.SetupBuildDirectory(rootBuildDir)
	if err != nil {
		log.Audit("The build directory could not be set up.")
		return err
	}

	// This needs to occur as early as possible so that the subsequent calls can use the log
	log.ConfigureGlobalLogger(filepath.Join(buildDir, buildLogFilename))

	if cmdErr := imageConfigDirExists(args.ConfigDir); cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	imageDefinition, cmdErr := parseImageDefinition(args.ConfigDir, args.DefinitionFile)
	if cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	ctx := buildContext(buildDir, combustionDir, args.ConfigDir, imageDefinition)

	if cmdErr = validateImageDefinition(ctx); cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	if err = appendKubernetesSELinuxRPMs(ctx); err != nil {
		log.Auditf("Configuring Kubernetes failed. %s", checkBuildLogMessage)
		zap.S().Fatalf("Failed to configure Kubernetes SELinux policy: %s", err)
	}

	appendElementalRPMs(ctx)

	appendHelm(ctx)

	if cmdErr = bootstrapDependencyServices(ctx, rootBuildDir); cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	defer func() {
		if r := recover(); r != nil {
			log.AuditInfo("Build failed unexpectedly, check the logs under the build directory for more information.")
			zap.S().Fatalf("Unexpected error occurred: %s", r)
		}
	}()

	builder := build.NewBuilder(ctx)
	if err = builder.Build(); err != nil {
		zap.S().Fatalf("An error occurred building the image: %s", err)
	}

	return nil
}

func imageConfigDirExists(configDir string) *cmd.Error {
	_, err := os.Stat(configDir)
	if err == nil {
		return nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		return &cmd.Error{
			UserMessage: fmt.Sprintf("The specified image configuration directory '%s' could not be found.", configDir),
		}
	}

	return &cmd.Error{
		UserMessage: fmt.Sprintf("Unable to check the filesystem for the image configuration directory '%s'.", configDir),
		LogMessage:  fmt.Sprintf("Reading image config dir failed: %v", err),
	}
}

func parseImageDefinition(configDir, definitionFile string) (*image.Definition, *cmd.Error) {
	definitionFilePath := filepath.Join(configDir, definitionFile)

	configData, err := os.ReadFile(definitionFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, &cmd.Error{
				UserMessage: fmt.Sprintf("The specified definition file '%s' could not be found.", definitionFilePath),
			}
		}

		return nil, &cmd.Error{
			UserMessage: fmt.Sprintf("The specified definition file '%s' could not be read.", definitionFilePath),
			LogMessage:  fmt.Sprintf("Reading definition file failed: %v", err),
		}
	}

	imageDefinition, err := image.ParseDefinition(configData)
	if err != nil {
		return nil, &cmd.Error{
			UserMessage: fmt.Sprintf("The image definition file '%s' could not be parsed.", definitionFilePath),
			LogMessage:  fmt.Sprintf("Parsing definition file failed: %v", err),
		}
	}

	return imageDefinition, nil
}

// Assembles the image build context with user-provided values and implementation defaults.
func buildContext(buildDir, combustionDir, configDir string, imageDefinition *image.Definition) *image.Context {
	ctx := &image.Context{
		ImageConfigDir:               configDir,
		BuildDir:                     buildDir,
		CombustionDir:                combustionDir,
		ImageDefinition:              imageDefinition,
		NetworkConfigGenerator:       network.ConfigGenerator{},
		NetworkConfiguratorInstaller: network.ConfiguratorInstaller{},
	}
	return ctx
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

	appendRPMs(ctx, image.AddRepo{URL: combustion.ElementalPackageRepository}, combustion.ElementalPackages...)
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

// If the image definition requires it, starts the necessary services, returning an error in the event of failure.
func bootstrapDependencyServices(ctx *image.Context, rootDir string) *cmd.Error {
	if !combustion.SkipRPMComponent(ctx) {
		p, err := podman.New(ctx.BuildDir)
		if err != nil {
			return &cmd.Error{
				UserMessage: "The services for RPM dependency resolution failed to start.",
				LogMessage:  fmt.Sprintf("Setting up Podman instance failed: %v", err),
			}
		}

		imgPath := filepath.Join(ctx.ImageConfigDir, "base-images", ctx.ImageDefinition.Image.BaseImage)
		imgType := ctx.ImageDefinition.Image.ImageType
		baseBuilder := resolver.NewTarballBuilder(ctx.BuildDir, imgPath, imgType, p)

		rpmResolver := resolver.New(ctx.BuildDir, p, baseBuilder, "")
		ctx.RPMResolver = rpmResolver
		ctx.RPMRepoCreator = rpm.NewRepoCreator(ctx.BuildDir)
	}

	if combustion.IsEmbeddedArtifactRegistryConfigured(ctx) {
		certsDir := filepath.Join(ctx.ImageConfigDir, combustion.K8sDir, combustion.HelmDir, combustion.CertsDir)
		ctx.HelmClient = helm.New(ctx.BuildDir, certsDir)
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		c, err := cache.New(rootDir)
		if err != nil {
			return &cmd.Error{
				UserMessage: "Setting up file caching failed.",
				LogMessage:  fmt.Sprintf("Initialising cache instance failed: %v", err),
			}
		}

		ctx.KubernetesScriptDownloader = kubernetes.ScriptDownloader{}
		ctx.KubernetesArtefactDownloader = kubernetes.ArtefactDownloader{
			Cache: c,
		}
	}

	return nil
}
