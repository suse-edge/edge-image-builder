package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/network"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	argConfigFile  = "config-file"
	argConfigDir   = "config-dir"
	argBuildDir    = "build-dir"
	argDeleteBuild = "delete-build-dir"
)

func processArgs() (*image.Context, error) {
	var (
		configFile     string
		configDir      string
		buildDir       string
		deleteBuildDir bool
	)

	flag.StringVar(&configFile, argConfigFile, "", "name of the image configuration file")
	flag.StringVar(&configDir, argConfigDir, "", "full path to the image configuration directory")
	flag.StringVar(&buildDir, argBuildDir, "", "full path to the directory to store build artifacts")
	flag.BoolVar(&deleteBuildDir, argDeleteBuild, false,
		"if specified, the build directory will be deleted after the image is built")
	flag.Parse()

	imageDefinition, err := parseImageDefinition(configFile, configDir)
	if err != nil {
		return nil, fmt.Errorf("parsing image definition file %s: %w", configFile, err)
	}

	err = validateImageConfigDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("validating the config dir %s: %w", configDir, err)
	}

	ctx, err := image.NewContext(configDir, buildDir, deleteBuildDir, imageDefinition, network.ConfigGenerator{}, network.ConfiguratorInstaller{})
	if err != nil {
		return nil, fmt.Errorf("building dir structure: %w", err)
	}

	setupLogging(ctx)

	return ctx, nil
}

func generateBuildLogFilename(ctx *image.Context) string {
	const buildLogFile = "eib-build-%s.log"

	timestamp := time.Now().Format("Jan02_15-04-05")
	filename := fmt.Sprintf(buildLogFile, timestamp)

	return filepath.Join(ctx.BuildDir, filename)
}

func setupLogging(ctx *image.Context) {
	logFilename := generateBuildLogFilename(ctx)

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

	builder := build.New(ctx)
	if err = builder.Build(); err != nil {
		zap.L().Fatal("An error occurred building the image", zap.Error(err))
	}

	if err = image.CleanUpBuildDir(ctx); err != nil {
		zap.L().Error("Failed to clean up build directory", zap.Error(err))
	}
}
