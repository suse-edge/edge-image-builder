package cmd

import (
	"fmt"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/urfave/cli/v2"
)

func validateGenerateFlags(c *cli.Context) error {
	outputType := strings.ToLower(c.String("output-type"))
	if outputType != image.TypeTar && outputType != image.TypeISO {
		return fmt.Errorf("invalid output-type '%s': must be either '%s' or '%s'", outputType, image.TypeTar, image.TypeISO)
	}

	generateArch := strings.ToLower(c.String("arch"))
	if generateArch != string(image.ArchTypeX86) && generateArch != string(image.ArchTypeARM) {
		return fmt.Errorf("invalid arch '%s': must be either '%s' or '%s'", generateArch, image.ArchTypeX86, image.ArchTypeARM)
	}

	cacheDirFlag := strings.ToLower(c.String("cache-dir"))
	cacheEnabledFlag := c.Bool("cache")
	err := validateCache(cacheDirFlag, cacheEnabledFlag)
	if err != nil {
		return err
	}

	return nil
}

func NewGenerateCommand(action func(*cli.Context) error) *cli.Command {
	return &cli.Command{
		Name:      "generate",
		Usage:     "Generate combustion drive",
		UsageText: fmt.Sprintf("%s generate [OPTIONS]", appName),
		Action:    action,
		Before:    validateGenerateFlags,
		Flags: []cli.Flag{
			DefinitionFileFlag,
			ConfigDirFlag,
			BuildDirFlag,
			CacheDirFlag,
			CacheFlag,
			&cli.StringFlag{
				Name:     "output-type",
				Usage:    "The desired output type",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Usage:    "The name of the file to generate",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "arch",
				Usage:    "The architecture of the generated artifacts",
				Required: true,
			},
		},
	}
}
