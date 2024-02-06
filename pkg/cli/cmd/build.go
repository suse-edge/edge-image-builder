package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

type BuildFlags struct {
	DefinitionFile string
	ConfigDir      string
	RootBuildDir   string
	Validate       bool
}

var BuildArgs BuildFlags

func NewBuildCommand(action func(*cli.Context) error) *cli.Command {
	return &cli.Command{
		Name:      "build",
		Usage:     "Build new image",
		UsageText: fmt.Sprintf("%s build [OPTIONS]", appName),
		Action:    action,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config-file",
				Usage:       "Name of the image definition file",
				Destination: &BuildArgs.DefinitionFile,
			},
			&cli.StringFlag{
				Name:        "config-dir",
				Usage:       "Full path to the image configuration directory",
				Required:    true,
				Destination: &BuildArgs.ConfigDir,
			},
			&cli.StringFlag{
				Name:        "build-dir",
				Usage:       "Full path to the directory to store build artifacts",
				Destination: &BuildArgs.RootBuildDir,
			},
			&cli.BoolFlag{
				Name:        "validate",
				Usage:       "If specified, the image definition will be validated but not built",
				Destination: &BuildArgs.Validate,
			},
		},
	}
}
