package build

import (
	"fmt"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

type configureCombustion func(ctx *image.Context) error

type Builder struct {
	context             *image.Context
	configureCombustion configureCombustion
}

func New(ctx *image.Context) *Builder {
	return &Builder{
		context:             ctx,
		configureCombustion: combustion.Configure,
	}
}

func (b *Builder) Build() error {
	log.Audit("Generating image customization components...")

	if err := b.configureCombustion(b.context); err != nil {
		return fmt.Errorf("configuring combustion: %w", err)
	}

	switch b.context.ImageDefinition.Image.ImageType {
	case image.TypeISO:
		log.Audit("Building ISO image...")
		if err := b.buildIsoImage(); err != nil {
			log.Audit("Error building ISO image, check the logs under the build directory for more information.")
			return err
		}
	case image.TypeRAW:
		log.Audit("Building RAW image...")
		if err := b.buildRawImage(); err != nil {
			log.Audit("Error building RAW image, check the logs under the build directory for more information.")
			return err
		}
	default:
		return fmt.Errorf("invalid imageType value specified, must be either \"%s\" or \"%s\"",
			image.TypeISO, image.TypeRAW)
	}

	log.Audit("Image build complete!")
	return nil
}

func (b *Builder) generateBuildDirFilename(filename string) string {
	return filepath.Join(b.context.BuildDir, filename)
}

func (b *Builder) generateOutputImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, b.context.ImageDefinition.Image.OutputImageName)
	return filename
}

func (b *Builder) generateBaseImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, "images", b.context.ImageDefinition.Image.BaseImage)
	return filename
}
