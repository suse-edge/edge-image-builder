package main

import (
	"flag"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/suse-edge/edge-image-builder/pkg/config"
)

const (
	argConfigFile = "config"
	argVerbose    = "verbose"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:    true,
		QuoteEmptyFields: true,
	})
	log.SetOutput(os.Stdout)
}

func processArgs() (*config.ImageConfig, error) {
	var (
		configFile string
		verbose    bool
	)

	flag.StringVar(&configFile, argConfigFile, "", "full path to the image configuration file")
	flag.BoolVar(&verbose, argVerbose, false, "enables extra logging information")
	flag.Parse()

	handleVerbose(verbose)
	imageConfig, err := handleImageConfig(configFile)

	return imageConfig, err
}

func handleVerbose(verbose bool) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
}

func handleImageConfig(configFile string) (*config.ImageConfig, error) {
	if configFile == "" {
		return nil, fmt.Errorf("the \"%s\" argument must be specified", argConfigFile)
	}

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

func main() {
	_, err := processArgs()
	if err != nil {
		log.Error(err)
	}

	// Call to building logic when it's finished
}
