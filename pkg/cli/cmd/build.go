package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

type BuildFlags struct {
	DefinitionFile string
	ConfigDir      string
	RootBuildDir   string
}

var BuildArgs BuildFlags

func NewBuildCommand(action func(*cli.Context) error) *cli.Command {
	return &cli.Command{
		Name:      "build",
		Usage:     "Build new image",
		UsageText: fmt.Sprintf("%s build [OPTIONS]", appName),
		Action:    action,
		Flags: []cli.Flag{
			DefinitionFileFlag,
			ConfigDirFlag,
			&cli.StringFlag{
				Name:        "build-dir",
				Usage:       "Full path to the directory to store build artifacts",
				Destination: &BuildArgs.RootBuildDir,
			},
		},
	}
}
