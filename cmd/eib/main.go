package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/suse-edge/edge-image-builder/pkg/build"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

const (
	argConfigFile = "config-file"
	argConfigDir  = "config-dir"
	argBuildDir   = "build-dir"
	argVerbose    = "verbose"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:    true,
		QuoteEmptyFields: true,
	})
	log.SetOutput(os.Stdout)
}

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

	handleVerbose(verbose)

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
		BuildDir: buildDir,
	}

	return imageConfig, &buildConfig, err
}

func handleVerbose(verbose bool) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
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
	ic, bc, err := processArgs()
	if err != nil {
		log.Error(err)
	}

	builder := build.New(ic, bc)
	err = builder.Build()
	if err != nil {
		log.Error(err)
	}
}
