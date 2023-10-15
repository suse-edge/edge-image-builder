package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/config"
	"go.uber.org/zap"
)

const (
	argConfigFile = "config-file"
	argConfigDir  = "config-dir"
	argBuildDir   = "build-dir"
	argVerbose    = "verbose"
)

func processArgs() (*config.ImageConfig, *config.BuildConfig, error) {
	var (
		configFile string
		configDir  string
		buildDir   string
		verbose    bool
	)

	flag.StringVar(&configFile, argConfigFile, "", "full path to the image configuration file")
	flag.StringVar(&configDir, argConfigDir, "", "full path to the image configuration directory")
	flag.StringVar(&buildDir, argBuildDir, "", "full path to the directory to store build artifacts")
	flag.BoolVar(&verbose, argVerbose, false, "enables extra logging information")
	flag.Parse()

	setupLogging(verbose)

	imageConfig, err := parseImageConfig(configFile)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing image config file %s: %w", configFile, err)
	}

	err = validateImageConfigDir(configDir)
	if err != nil {
		return nil, nil, fmt.Errorf("validating the config dir %s: %w", configDir, err)
	}
	buildConfig := config.BuildConfig{
		ImageConfigDir: configDir,
		BuildDir:       buildDir,
	}

	return imageConfig, &buildConfig, err
}

func setupLogging(verbose bool) {
	logLevel := zap.InfoLevel
	if verbose {
		logLevel = zap.DebugLevel
	}

	logConfig := zap.Config{
		Level: zap.NewAtomicLevelAt(logLevel),
	}

	logger := zap.Must(logConfig.Build())

	// Set our configured logger to be accessed globally by zap.L()
	zap.ReplaceGlobals(logger)
}

func parseImageConfig(configFile string) (*config.ImageConfig, error) {
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("image configuration file \"%s\" cannot be read: %s", configFile, err)
	}

	imageConfig, err := config.Parse(configData)
	if err != nil {
		return nil, fmt.Errorf("error parsing configuration file \"%s\": %s", configFile, err)
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
	imageConfig, buildConfig, err := processArgs()
	if err != nil {
		zap.L().Error("parsing CLI arguments", zap.Error(err))
	}

	builder := build.New(imageConfig, buildConfig)
	err = builder.Build()
	if err != nil {
		zap.L().Error("building the image", zap.Error(err))
	}
}
