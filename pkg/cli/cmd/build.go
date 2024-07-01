package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/suse-edge/edge-image-builder/pkg/env"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"gopkg.in/yaml.v3"
)

type BuildFlags struct {
	DefinitionFile string
	ConfigDir      string
	RootBuildDir   string
}

var BuildArgs BuildFlags
var ArtifactSources env.ArtifactSources

func NewBuildCommand(action func(*cli.Context) error) *cli.Command {
	buildFlags := []cli.Flag{
		DefinitionFileFlag,
		ConfigDirFlag,
		&cli.StringFlag{
			Name:        "build-dir",
			Usage:       "Full path to the directory to store build artifacts",
			Destination: &BuildArgs.RootBuildDir,
		},
	}

	artifactFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "metallb.chart",
			Destination: &ArtifactSources.MetalLB.Chart,
			Usage:       "Name of the MetalLB Helm chart",
			Category:    "Artifact sources",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "metallb.repository",
			Destination: &ArtifactSources.MetalLB.Repository,
			Usage:       "Address of the MetalLB Helm repository",
			Category:    "Artifact sources",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "metallb.version",
			Destination: &ArtifactSources.MetalLB.Version,
			Usage:       "Version of the MetalLB Helm chart",
			Category:    "Artifact sources",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "endpoint-copier-operator.chart",
			Destination: &ArtifactSources.EndpointCopierOperator.Chart,
			Usage:       "Name of the Endpoint Copier Operator Helm chart",
			Category:    "Artifact sources",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "endpoint-copier-operator.repository",
			Destination: &ArtifactSources.EndpointCopierOperator.Repository,
			Usage:       "Address of the Endpoint Copier Operator Helm repository",
			Category:    "Artifact sources",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "endpoint-copier-operator.version",
			Destination: &ArtifactSources.EndpointCopierOperator.Version,
			Usage:       "Version of the Endpoint Copier Operator Helm chart",
			Category:    "Artifact sources",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "elemental.register-repository",
			Destination: &ArtifactSources.Elemental.RegisterRepository,
			Usage:       "Address of the elemental-register RPM repository",
			Category:    "Artifact sources",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "elemental.system-agent-repository",
			Destination: &ArtifactSources.Elemental.SystemAgentRepository,
			Usage:       "Address of the elemental-system-agent RPM repository",
			Category:    "Artifact sources",
		}),
	}

	return &cli.Command{
		Name:      "build",
		Usage:     "Build new image",
		UsageText: fmt.Sprintf("%s build [OPTIONS]", appName),
		Action:    action,
		Before:    parseArtifactSources(artifactFlags),
		Flags:     append(buildFlags, artifactFlags...),
	}
}

func parseArtifactSources(flags []cli.Flag) func(*cli.Context) error {
	const artifactsConfigFile = "artifacts.yaml"

	return func(ctx *cli.Context) error {
		b, err := os.ReadFile(artifactsConfigFile)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("artifact sources file '%s' does not exist", artifactsConfigFile)
			}

			return fmt.Errorf("reading artifact sources file: %w", err)
		}

		var sources map[any]any
		if err = yaml.Unmarshal(b, &sources); err != nil {
			return fmt.Errorf("decoding artifacts sources: %w", err)
		}

		inputSource := altsrc.NewMapInputSource(artifactsConfigFile, sources)
		return altsrc.ApplyInputSourceValues(ctx, inputSource, flags)
	}
}
