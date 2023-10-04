package main

import (
	"bytes"
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

func parseFlags(programName string, args []string) (imageConfig *config.ImageConfig, output string, err error) {
	flags := flag.NewFlagSet(programName, flag.ContinueOnError)
	var buf bytes.Buffer
	flags.SetOutput(&buf)

	var (
		configFile string
		verbose    bool
	)

	flags.StringVar(&configFile, argConfigFile, "", "full path to the image configuration file")
	flags.BoolVar(&verbose, argVerbose, false, "enables extra logging information")

	err = flags.Parse(args)

	// Help
	if err == flag.ErrHelp {
		return nil, buf.String(), err
	}

	// Verbose
	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	// Image Configuration File
	if configFile == "" {
		return nil, buf.String(), fmt.Errorf("the \"%s\" argument must be specified", argConfigFile)
	}

	info, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		return nil, buf.String(), fmt.Errorf("image configuration file \"%s\" does not exist", configFile)
	}
	if info.IsDir() {
		return nil, buf.String(), fmt.Errorf("image configuration file \"%s\" cannot be a directory", configFile)
	}

	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, buf.String(), fmt.Errorf("image configuration file \"%s\" cannot be read: %s", configFile, err)
	}

	_, err = config.Parse(configData)
	if err != nil {
		return nil, buf.String(), fmt.Errorf("error parsing configuration file \"%s\": %s", configFile, err)
	}

	return imageConfig, buf.String(), nil
}

func main() {
	_, output, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		fmt.Println(output)
		os.Exit(2)
	} else if err != nil {
		log.Fatal(err)
	}
}
