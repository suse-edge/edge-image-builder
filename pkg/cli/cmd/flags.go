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
	GenerateDefinitionFileFlag = &cli.StringFlag{
		Name:        "definition-file",
		Usage:       "Name of the image definition file",
		Destination: &GenerateArgs.GenerateDefinitionFile,
	}
	GenerateConfigDirFlag = &cli.StringFlag{
		Name:        "config-dir",
		Usage:       "Full path to the image configuration directory",
		Value:       "/eib",
		Destination: &GenerateArgs.GenerateConfigDir,
	}
	GenerateOutputTypeFlag = &cli.StringFlag{
		Name:        "output-type",
		Usage:       "The desired output type",
		Required:    true,
		Destination: &GenerateArgs.GenerateOutputType,
	}
	GenerateOutputFlag = &cli.StringFlag{
		Name:        "output",
		Usage:       "The name of the file to generate",
		Required:    true,
		Destination: &GenerateArgs.GenerateOutput,
	}
	GenerateOutputArch = &cli.StringFlag{
		Name:        "arch",
		Usage:       "The architecture of the generated artifacts",
		Required:    true,
		Destination: &GenerateArgs.GenerateArch,
	}
)
