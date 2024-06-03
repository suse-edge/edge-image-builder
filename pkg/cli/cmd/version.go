package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func NewVersionCommand(action func(*cli.Context) error) *cli.Command {
	return &cli.Command{
		Name:      "version",
		Usage:     "Inspect program version",
		UsageText: fmt.Sprintf("%s version", appName),
		Action:    action,
	}
}
