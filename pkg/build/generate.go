package build

import (
	"fmt"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

type Generator struct {
	context           *image.Context
	imageConfigurator imageConfigurator
}

func NewGenerator(ctx *image.Context, imageConfigurator imageConfigurator) *Generator {
	return &Generator{
		context:           ctx,
		imageConfigurator: imageConfigurator,
	}
}

func (g *Generator) Generate() error {
	log.Audit("Generating image customization components...")

	if err := g.imageConfigurator.Configure(g.context); err != nil {
		log.Audit("Error configuring customization components.")
		return fmt.Errorf("configuring image: %w", err)
	}

	switch g.context.ImageDefinition.Image.ImageType {
	case image.TypeTar:
		log.Audit("Generating combustion tarball...")
		if err := g.generateTarball(); err != nil {
			log.Audit("Error generating combustion tarball.")
			return err
		}
	case image.TypeISO:
		log.Audit("Generating combustion iso...")
		if err := g.generateCombustionISO(); err != nil {
			log.Audit("Error generating combustion iso.")
			return err
		}
	default:
		return fmt.Errorf("invalid output type specified, must be either \"%s\" or \"%s\"",
			image.TypeISO, image.TypeTar)
	}

	log.Auditf("Config drive generation complete, it can be found at: %s",
		g.context.ImageDefinition.Image.OutputImageName)
	return nil
}
