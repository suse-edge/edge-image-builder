package cmd

import "github.com/urfave/cli/v2"

type CommonFlags struct {
	DefinitionFile string
	ConfigDir      string
	RootBuildDir   string
}

var CommonArgs CommonFlags

var (
	DefinitionFileFlag = &cli.StringFlag{
		Name:        "definition-file",
		Usage:       "Name of the image definition file",
		Destination: &CommonArgs.DefinitionFile,
	}
	ConfigDirFlag = &cli.StringFlag{
		Name:        "config-dir",
		Usage:       "Full path to the image configuration directory",
		Value:       "/eib",
		Destination: &CommonArgs.ConfigDir,
	}
	BuildDirFlag = &cli.StringFlag{
		Name:        "build-dir",
		Usage:       "Full path to the directory to store build artifacts",
		Destination: &CommonArgs.RootBuildDir,
	}
)
