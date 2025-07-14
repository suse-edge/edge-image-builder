package cmd

import "github.com/urfave/cli/v2"

var (
	DefinitionFileFlag = &cli.StringFlag{
		Name:        "definition-file",
		Usage:       "Name of the image definition file",
		Destination: &BuildArgs.DefinitionFile,
	}
	ConfigDirFlag = &cli.StringFlag{
		Name:        "config-dir",
		Usage:       "Full path to the image configuration directory",
		Value:       "/eib",
		Destination: &BuildArgs.ConfigDir,
	}
	OutputFlag = &cli.StringFlag{
		Name:        "output-type",
		Usage:       "The desired output type",
		Required:    true,
		Destination: &GenerateArgs.OutputType,
	}
)
