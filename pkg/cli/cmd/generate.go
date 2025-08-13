package cmd

import (
	"fmt"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/urfave/cli/v2"
)

type GenerateFlags struct {
	GenerateDefinitionFile string
	GenerateConfigDir      string
	GenerateRootBuildDir   string
	GenerateOutputType     string
	GenerateArch           string
	GenerateOutput         string
}

var GenerateArgs GenerateFlags

func validateGenerateFlags(c *cli.Context) error {
	outputType := strings.ToLower(c.String("output-type"))
	if outputType != image.TypeTar && outputType != image.TypeISO {
		return fmt.Errorf("invalid output-type '%s': must be either '%s' or '%s'", outputType, image.TypeTar, image.TypeISO)
	}

	generateArch := strings.ToLower(c.String("arch"))
	if generateArch != string(image.ArchTypeX86) && generateArch != string(image.ArchTypeARM) {
		return fmt.Errorf("invalid arch '%s': must be either '%s' or '%s'", generateArch, image.ArchTypeX86, image.ArchTypeARM)
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
			GenerateDefinitionFileFlag,
			GenerateConfigDirFlag,
			GenerateOutputFlag,
			GenerateOutputArch,
			GenerateOutputTypeFlag,
			&cli.StringFlag{
				Name:        "build-dir",
				Usage:       "Full path to the directory to store build artifacts",
				Destination: &GenerateArgs.GenerateRootBuildDir,
			},
		},
	}
}
