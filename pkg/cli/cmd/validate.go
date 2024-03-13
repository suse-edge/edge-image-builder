package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func NewValidateCommand(action func(*cli.Context) error) *cli.Command {
	return &cli.Command{
		Name:      "validate",
		Usage:     "Validate image configuration",
		UsageText: fmt.Sprintf("%s validate [OPTIONS]", appName),
		Action:    action,
		Flags: []cli.Flag{
			DefinitionFileFlag,
			ConfigDirFlag,
		},
	}
}
