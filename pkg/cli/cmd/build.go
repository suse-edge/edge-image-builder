package cmd

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
)

func validateBuildFlags(c *cli.Context) error {
	cacheDirFlag := strings.ToLower(c.String("cache-dir"))
	cacheEnabledFlag := c.Bool("cache")

	return validateCache(cacheDirFlag, cacheEnabledFlag)
}

func validateCache(cacheDir string, cacheEnabled bool) error {
	if !cacheEnabled {
		if cacheDir != "/eib-cache" {
			return fmt.Errorf("`cache-dir` cannot be specified when `cache` is set to false")
		}
	}

	return nil
}

func NewBuildCommand(action func(*cli.Context) error) *cli.Command {
	return &cli.Command{
		Name:      "build",
		Usage:     "Build new image",
		UsageText: fmt.Sprintf("%s build [OPTIONS]", appName),
		Before:    validateBuildFlags,
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
