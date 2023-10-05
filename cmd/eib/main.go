package main

import (
	"flag"
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

func processArgs() (imageConfig *config.ImageConfig) {
	var (
		configFile string
		verbose    bool
	)

	flag.StringVar(&configFile, argConfigFile, "", "full path to the image configuration file")
	flag.BoolVar(&verbose, argVerbose, false, "enables extra logging information")
	flag.Parse()

	handleVerbose(verbose)
	imageConfig = handleImageConfig(configFile)

	return imageConfig
}

func handleVerbose(verbose bool) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
}

func handleImageConfig(configFile string) *config.ImageConfig {
	if configFile == "" {
		log.Fatalf("the \"%s\" argument must be specified", argConfigFile)
	}

	info, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		log.Fatalf("image configuration file \"%s\" does not exist", configFile)
	}
	if info.IsDir() {
		log.Fatalf("image configuration file \"%s\" cannot be a directory", configFile)
	}

	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("image configuration file \"%s\" cannot be read: %s", configFile, err)
	}

	imageConfig, err := config.Parse(configData)
	if err != nil {
		log.Fatalf("error parsing configuration file \"%s\": %s", configFile, err)
	}

	return imageConfig
}

func main() {
	processArgs()
}
