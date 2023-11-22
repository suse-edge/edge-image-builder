package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	argConfigFile  = "config-file"
	argConfigDir   = "config-dir"
	argBuildDir    = "build-dir"
	argDeleteBuild = "delete-build-dir"
	argVerbose     = "verbose"
)

func processArgs() (*image.Definition, *image.Context, error) {
	var (
		configFile     string
		configDir      string
		buildDir       string
		deleteBuildDir bool
		verbose        bool
	)

	flag.StringVar(&configFile, argConfigFile, "", "name of the image configuration file")
	flag.StringVar(&configDir, argConfigDir, "", "full path to the image configuration directory")
	flag.StringVar(&buildDir, argBuildDir, "", "full path to the directory to store build artifacts")
	flag.BoolVar(&deleteBuildDir, argDeleteBuild, false,
		"if specified, the build directory will be deleted after the image is built")
	flag.BoolVar(&verbose, argVerbose, false, "enables extra logging information")
	flag.Parse()

	setupLogging(verbose)

	imageDefinition, err := parseImageDefinition(configFile, configDir)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing image config file %s: %w", configFile, err)
	}

	err = validateImageConfigDir(configDir)
	if err != nil {
		return nil, nil, fmt.Errorf("validating the config dir %s: %w", configDir, err)
	}

	ctx, err := image.NewContext(configDir, buildDir, deleteBuildDir, imageDefinition)
	if err != nil {
		return nil, nil, fmt.Errorf("building dir structure: %w", err)
	}

	return imageDefinition, ctx, err
}

func setupLogging(verbose bool) {
	logLevel := zap.InfoLevel
	if verbose {
		logLevel = zap.DebugLevel
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	logConfig := zap.Config{
		Level:         zap.NewAtomicLevelAt(logLevel),
		Encoding:      "console",
		EncoderConfig: encoderCfg,
		OutputPaths: []string{
			"stdout",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
	}

	logger := zap.Must(logConfig.Build())

	// Set our configured logger to be accessed globally by zap.L()
	zap.ReplaceGlobals(logger)
}

func parseImageDefinition(configFile string, configDir string) (*image.Definition, error) {
	configFilePath := filepath.Join(configDir, configFile)
	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("image configuration file \"%s\" cannot be read: %w", configFile, err)
	}

	imageConfig, err := image.ParseDefinition(configData)
	if err != nil {
		return nil, fmt.Errorf("error parsing configuration file \"%s\": %w", configFile, err)
	}

	return imageConfig, nil
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
	imageConfig, ctx, err := processArgs()
	if err != nil {
		zap.L().Fatal("CLI arguments could not be parsed", zap.Error(err))
	}

	builder := build.New(imageConfig, ctx, combustion.Configure)
	if err = builder.Build(); err != nil {
		zap.L().Fatal("An error occurred building the image", zap.Error(err))
	}

	if err = image.CleanUpBuildDir(ctx); err != nil {
		zap.L().Error("Failed to clean up build directory", zap.Error(err))
	}
}
