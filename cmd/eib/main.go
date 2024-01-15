package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/kubernetes"
	audit "github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/network"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/rpm/resolver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	argConfigFile = "config-file"
	argConfigDir  = "config-dir"
	argBuildDir   = "build-dir"
)

func processArgs() (*image.Context, error) {
	var (
		configFile   string
		configDir    string
		rootBuildDir string
	)

	flag.StringVar(&configFile, argConfigFile, "", "name of the image configuration file")
	flag.StringVar(&configDir, argConfigDir, "", "full path to the image configuration directory")
	flag.StringVar(&rootBuildDir, argBuildDir, "", "full path to the directory to store build artifacts")
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

	if !combustion.SkipRPMComponent(ctx) {
		p, err := podman.New(buildDir)
		if err != nil {
			return nil, fmt.Errorf("starting podman client: %w", err)
		}

		imgPath := filepath.Join(configDir, "images", imageDefinition.Image.BaseImage)
		rpmResolver := resolver.New(buildDir, imgPath, imageDefinition.Image.ImageType, p)
		ctx.RPMResolver = rpmResolver
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

	err = image.ValidateDefinition(imageDefinition)
	if err != nil {
		return nil, fmt.Errorf("error validating definition file: %w", err)
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
