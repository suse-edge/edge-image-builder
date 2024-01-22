package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/build"
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
	argConfigFile = "config-file"
	argConfigDir  = "config-dir"
	argBuildDir   = "build-dir"
	argValidate   = "validate"
)

func processArgs() (*image.Context, error) {
	var (
		configFile   string
		configDir    string
		rootBuildDir string
		validate     bool
	)

	flag.StringVar(&configFile, argConfigFile, "", "name of the image configuration file")
	flag.StringVar(&configDir, argConfigDir, "", "full path to the image configuration directory")
	flag.StringVar(&rootBuildDir, argBuildDir, "", "full path to the directory to store build artifacts")
	flag.BoolVar(&validate, argValidate, false, "if specified, the image definition will be validated but not built")
	flag.Parse()

	imageDefinition, err := parseImageDefinition(configFile, configDir)
	if err != nil {
		return nil, fmt.Errorf("parsing image definition file %s: %w", configFile, err)
	}

	err = validateImageConfigDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("validating the config dir %s: %w", configDir, err)
	}

	buildDir, combustionDir, err := build.SetupBuildDirectory(rootBuildDir)
	if err != nil {
		return nil, fmt.Errorf("setting up build directory: %w", err)
	}

	setupLogging(buildDir)

	ctx := &image.Context{
		ImageConfigDir:               configDir,
		BuildDir:                     buildDir,
		CombustionDir:                combustionDir,
		ImageDefinition:              imageDefinition,
		NetworkConfigGenerator:       network.ConfigGenerator{},
		NetworkConfiguratorInstaller: network.ConfiguratorInstaller{},
		KubernetesScriptInstaller:    kubernetes.ScriptInstaller{},
		KubernetesArtefactDownloader: kubernetes.ArtefactDownloader{},
	}

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
			zap.S().Fatalf("Image definition validation failures:\n%s", s)
		}
	}

	if validate {
		audit.Audit("The specified image definition is valid.")
		os.Exit(0)
	}

	if !combustion.SkipRPMComponent(ctx) {
		p, err := podman.New(buildDir)
		if err != nil {
			return nil, fmt.Errorf("starting podman client: %w", err)
		}

		imgPath := filepath.Join(configDir, "images", imageDefinition.Image.BaseImage)
		rpmResolver := resolver.New(buildDir, imgPath, imageDefinition.Image.ImageType, p)
		ctx.RPMResolver = rpmResolver
		ctx.RPMRepoCreator = rpm.NewRepoCreator(buildDir)
	}

	return ctx, nil
}

func setupLogging(buildDir string) {
	logFilename := filepath.Join(buildDir, "eib-build.log")

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

func parseImageDefinition(configFile string, configDir string) (*image.Definition, error) {
	configFilePath := filepath.Join(configDir, configFile)
	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("image definition file \"%s\" cannot be read: %w", configFile, err)
	}

	imageDefinition, err := image.ParseDefinition(configData)
	if err != nil {
		return nil, fmt.Errorf("error parsing definition file \"%s\": %w", configFile, err)
	}

	return imageDefinition, nil
}

func validateImageConfigDir(configDir string) error {
	if configDir == "" {
		return fmt.Errorf("-%s must be specified", argConfigDir)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return err
	}
	return nil
}

func main() {
	ctx, err := processArgs()
	if err != nil {
		// use standard logger, zap is not yet configured
		log.Fatalf("CLI arguments could not be parsed: %s", err)
	}

	defer func() {
		if r := recover(); r != nil {
			audit.Audit("Build failed unexpectedly, check the logs under the build directory for more information.")
			zap.S().Fatalf("Unexpected error occurred: %s", r)
		}
	}()

	builder := build.NewBuilder(ctx)
	if err = builder.Build(); err != nil {
		zap.S().Fatalf("An error occurred building the image: %s", err)
	}
}
