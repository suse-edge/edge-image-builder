package cmd

import (
	"fmt"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/urfave/cli/v2"
	"strings"
)

type GenerateFlags struct {
	DefinitionFile string
	ConfigDir      string
	RootBuildDir   string
	OutputType     string
}

var GenerateArgs GenerateFlags

func validateGenerateFlags(c *cli.Context) error {
	outputType := strings.ToLower(c.String("output-type"))
	if outputType != image.TypeTar && outputType != image.TypeCombustionIso {
		return fmt.Errorf("invalid output-type '%s': must be either 'combustion-iso' or 'tar'", outputType)
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
			OutputFlag,
			&cli.StringFlag{
				Name:        "build-dir",
				Usage:       "Full path to the directory to store build artifacts",
				Destination: &GenerateArgs.RootBuildDir,
			},
		},
	}
}
