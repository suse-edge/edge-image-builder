package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/cache"
	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/image/validation"
	"github.com/suse-edge/edge-image-builder/pkg/kubernetes"
	audit "github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/network"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/rpm/resolver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	argDefinitionFile = "config-file"
	argConfigDir      = "config-dir"
	argBuildDir       = "build-dir"
	argValidate       = "validate"
)

const (
	logFilename     = "eib-build.log"
	checkLogMessage = "Please check the eib-build.log file under the build directory for more information."
)

type CLIArguments struct {
	definitionFile string
	configDir      string
	rootBuildDir   string
	validate       bool
}

func main() {
	cliArguments := parseCliArguments()

	buildDir, combustionDir, err := build.SetupBuildDirectory(cliArguments.rootBuildDir)
	if err != nil {
		audit.Auditf("The build directory could not be setup under the configuration directory '%s'.", cliArguments.configDir)
		audit.AuditInfo(err.Error())
		os.Exit(1)
	}

	// This needs to occur as early as possible so that the subsequent calls can use the log
	setupLogging(buildDir)

	configDirExists := doesImageConfigDirExist(cliArguments)
	if !configDirExists {
		os.Exit(1)
	}

	imageDefinition := parseImageDefinition(cliArguments)
	if imageDefinition == nil {
		os.Exit(1)
	}

	ctx := buildContext(buildDir, combustionDir, cliArguments.configDir, imageDefinition)

	isDefinitionValid := isImageDefinitionValid(ctx)
	if !isDefinitionValid {
		os.Exit(1)
	}

	if cliArguments.validate {
		// If we got this far, the image is valid. If we're in this block, the user wants execution to stop.
		audit.AuditInfo("The specified image definition is valid.")
		os.Exit(0)
	}

	if !bootstrapDependencyServices(ctx, cliArguments.rootBuildDir) {
		os.Exit(1)
	}

	defer func() {
		if r := recover(); r != nil {
			audit.AuditInfo("Build failed unexpectedly, check the logs under the build directory for more information.")
			zap.S().Fatalf("Unexpected error occurred: %s", r)
		}
	}()

	builder := build.NewBuilder(ctx)
	if err := builder.Build(); err != nil {
		zap.S().Fatalf("An error occurred building the image: %s", err)
	}
}

// Extract the user's CLI arguments into their own struct.
func parseCliArguments() CLIArguments {
	cliArguments := CLIArguments{}

	flag.StringVar(&cliArguments.definitionFile, argDefinitionFile, "", "name of the image definition file")
	flag.StringVar(&cliArguments.configDir, argConfigDir, "", "full path to the image configuration directory")
	flag.StringVar(&cliArguments.rootBuildDir, argBuildDir, "", "full path to the directory to store build artifacts")
	flag.BoolVar(&cliArguments.validate, argValidate, false, "if specified, the image definition will be validated but not built")
	flag.Parse()

	return cliArguments
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

// Returns whether the image configuration directory was specified and can be read, displaying
// the appropriate messages to the user. Returns 'true' if the directory exists and execution can proceed,
// 'false' otherwise.
func doesImageConfigDirExist(cliArguments CLIArguments) bool {
	configDir := cliArguments.configDir

	if configDir == "" {
		audit.AuditInfof("The '%s' argument must be specified.", argConfigDir)
		return false
	}

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
func parseImageDefinition(cliArguments CLIArguments) *image.Definition {
	definitionFilePath := filepath.Join(cliArguments.configDir, cliArguments.definitionFile)

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
