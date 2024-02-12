package build

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/cache"
	"github.com/suse-edge/edge-image-builder/pkg/cli/cmd"
	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/image/validation"
	"github.com/suse-edge/edge-image-builder/pkg/kubernetes"
	audit "github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/network"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/rpm/resolver"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logFilename     = "eib-build.log"
	checkLogMessage = "Please check the eib-build.log file under the build directory for more information."
)

func Run(*cli.Context) error {
	args := &cmd.BuildArgs

	buildDir, combustionDir, err := build.SetupBuildDirectory(args.RootBuildDir)
	if err != nil {
		audit.Auditf("The build directory could not be setup under the configuration directory '%s'.", args.ConfigDir)
		return err
	}

	// This needs to occur as early as possible so that the subsequent calls can use the log
	setupLogging(buildDir)

	configDirExists := doesImageConfigDirExist(args.ConfigDir)
	if !configDirExists {
		os.Exit(1)
	}

	imageDefinition := parseImageDefinition(args.ConfigDir, args.DefinitionFile)
	if imageDefinition == nil {
		os.Exit(1)
	}

	ctx := buildContext(buildDir, combustionDir, args.ConfigDir, imageDefinition)

	isDefinitionValid := isImageDefinitionValid(ctx)
	if !isDefinitionValid {
		os.Exit(1)
	}

	if args.Validate {
		// If we got this far, the image is valid. If we're in this block, the user wants execution to stop.
		audit.AuditInfo("The specified image definition is valid.")
		return nil
	}

	appendElementalRPMs(ctx)

	if !bootstrapDependencyServices(ctx, args.RootBuildDir) {
		os.Exit(1)
	}

	defer func() {
		if r := recover(); r != nil {
			audit.AuditInfo("Build failed unexpectedly, check the logs under the build directory for more information.")
			zap.S().Fatalf("Unexpected error occurred: %s", r)
		}
	}()

	builder := build.NewBuilder(ctx)
	if err = builder.Build(); err != nil {
		zap.S().Fatalf("An error occurred building the image: %s", err)
	}

	return nil
}

// Configures the global logger.
func setupLogging(buildDir string) {
	logFilename := filepath.Join(buildDir, logFilename)

	logConfig := zap.NewProductionConfig()
	logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logConfig.Encoding = "console"
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logConfig.OutputPaths = []string{logFilename}

	logger := zap.Must(logConfig.Build())

	// Set our configured logger to be accessed globally by zap.L()
	zap.ReplaceGlobals(logger)
}

// Returns whether the image configuration directory can be read, displaying
// the appropriate messages to the user. Returns 'true' if the directory exists and execution can proceed,
// 'false' otherwise.
func doesImageConfigDirExist(configDir string) bool {
	_, err := os.Stat(configDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			audit.AuditInfof("The specified image configuration directory '%s' could not be found.", configDir)
			return false
		}
		audit.AuditInfof("Unable to check the filesystem for the image configuration directory '%s'. %s",
			configDir, checkLogMessage)
		zap.S().Error(err)
		return false
	}

	return true
}

// Attempts to parse the specified image definition file, displaying the appropriate messages to the user.
// Returns a populated `image.Context` struct if successful, `nil` if the definition could not be parsed.
func parseImageDefinition(configDir, definitionFile string) *image.Definition {
	definitionFilePath := filepath.Join(configDir, definitionFile)

	configData, err := os.ReadFile(definitionFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			audit.AuditInfof("The specified definition file '%s' could not be found.", definitionFilePath)
		} else {
			audit.AuditInfof("The specified definition file '%s' could not be read. %s", definitionFilePath, checkLogMessage)
			zap.S().Error(err)
		}
		return nil
	}

	imageDefinition, err := image.ParseDefinition(configData)
	if err != nil {
		audit.AuditInfof("The image definition file '%s' could not be parsed. %s", definitionFilePath, checkLogMessage)
		zap.S().Error(err)
		return nil
	}

	return imageDefinition
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

// Runs the image definition validation, displaying the appropriate messages to the user in the event
// of a failure. Returns 'true' if the definition is valid; 'false' otherwise.
func isImageDefinitionValid(ctx *image.Context) bool {
	failedValidations := validation.ValidateDefinition(ctx)
	if len(failedValidations) > 0 {
		audit.Audit("Image definition validation found the following errors:")

		logMessageBuilder := strings.Builder{}

		orderedComponentNames := make([]string, 0, len(failedValidations))
		for c := range failedValidations {
			orderedComponentNames = append(orderedComponentNames, c)
		}
		slices.Sort(orderedComponentNames)

		for _, componentName := range orderedComponentNames {
			failures := failedValidations[componentName]
			audit.Audit(fmt.Sprintf("  %s", componentName))
			for _, cf := range failures {
				audit.Audit(fmt.Sprintf("    %s", cf.UserMessage))
				logMessageBuilder.WriteString(cf.UserMessage + "\n")
				if cf.Error != nil {
					logMessageBuilder.WriteString("\t" + cf.Error.Error() + "\n")
				}
			}
		}

		if s := logMessageBuilder.String(); s != "" {
			zap.S().Errorf("image definition validation failures:\n%s", s)
		}

		audit.AuditInfo(checkLogMessage)

		return false
	}

	return true
}

func appendElementalRPMs(ctx *image.Context) {
	elementalDir := filepath.Join(ctx.ImageConfigDir, "elemental")
	if _, err := os.Stat(elementalDir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.S().Warnf("Looking for 'elemental' dir failed unexpectedly: %s", err)
		}

		return
	}

	audit.Audit("Elemental registration is configured. The necessary RPM packages will be downloaded.")
	zap.S().Info("Elemental registration requested")

	packageList := ctx.ImageDefinition.OperatingSystem.Packages.PKGList
	packageList = append(packageList, "elemental-register", "elemental-system-agent")

	repositories := ctx.ImageDefinition.OperatingSystem.Packages.AdditionalRepos
	repositories = append(repositories,
		image.AddRepo{URL: combustion.ElementalPackageRepository})

	ctx.ImageDefinition.OperatingSystem.Packages.PKGList = packageList
	ctx.ImageDefinition.OperatingSystem.Packages.AdditionalRepos = repositories
}

// If the image definition requires it, starts the necessary services, displaying appropriate messages
// to users in the event of an error. Returns 'true' if execution should proceed given that all dependencies
// are satisfied; 'false' otherwise.
func bootstrapDependencyServices(ctx *image.Context, rootDir string) bool {
	if !combustion.SkipRPMComponent(ctx) {
		p, err := podman.New(ctx.BuildDir)
		if err != nil {
			audit.AuditInfof("The services for RPM dependency resolution failed to start. %s", checkLogMessage)
			zap.S().Error(err)
			return false
		}

		imgPath := filepath.Join(ctx.ImageConfigDir, "base-images", ctx.ImageDefinition.Image.BaseImage)
		rpmResolver := resolver.New(ctx.BuildDir, imgPath, ctx.ImageDefinition.Image.ImageType, p)
		ctx.RPMResolver = rpmResolver
		ctx.RPMRepoCreator = rpm.NewRepoCreator(ctx.BuildDir)
	}

	if ctx.ImageDefinition.Kubernetes.Version != "" {
		c, err := cache.New(rootDir)
		if err != nil {
			audit.AuditInfof("Failed to initialise file caching. %s", checkLogMessage)
			zap.S().Error(err)
			return false
		}

		ctx.KubernetesScriptInstaller = kubernetes.ScriptInstaller{}
		ctx.KubernetesArtefactDownloader = kubernetes.ArtefactDownloader{
			Cache: c,
		}
	}

	return true
}
