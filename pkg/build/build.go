package build

import (
	"fmt"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/image"
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
	if err := b.configureCombustion(b.context); err != nil {
		return fmt.Errorf("configuring combustion: %w", err)
	}

	switch b.context.ImageDefinition.Image.ImageType {
	case image.TypeISO:
		return b.buildIsoImage()
	case image.TypeRAW:
		return b.buildRawImage()
	default:
		return fmt.Errorf("invalid imageType value specified, must be either \"%s\" or \"%s\"",
			image.TypeISO, image.TypeRAW)
	}
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
