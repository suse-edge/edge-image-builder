package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func NewBuildCommand(action func(*cli.Context) error) *cli.Command {
	return &cli.Command{
		Name:      "build",
		Usage:     "Build new image",
		UsageText: fmt.Sprintf("%s build [OPTIONS]", appName),
		Action:    action,
		Flags: []cli.Flag{
			DefinitionFileFlag,
			ConfigDirFlag,
			BuildDirFlag,
			CacheDirFlag,
			CacheFlag,
		},
	}
}
