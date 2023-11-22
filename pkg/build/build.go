package build

import (
	"fmt"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type configureCombustion func(ctx *image.Context) error

type Builder struct {
	imageDefinition     *image.Definition
	context             *image.Context
	configureCombustion configureCombustion
}

func New(imageDefinition *image.Definition, ctx *image.Context, configureCombustionFunc configureCombustion) *Builder {
	return &Builder{
		imageDefinition:     imageDefinition,
		context:             ctx,
		configureCombustion: configureCombustionFunc,
	}
}

func (b *Builder) Build() error {
	if err := b.configureCombustion(b.context); err != nil {
		return fmt.Errorf("configuring combustion: %w", err)
	}

	switch b.imageDefinition.Image.ImageType {
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
	filename := filepath.Join(b.context.ImageConfigDir, b.imageDefinition.Image.OutputImageName)
	return filename
}

func (b *Builder) generateBaseImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, "images", b.imageDefinition.Image.BaseImage)
	return filename
}
