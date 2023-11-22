package build

import (
	"fmt"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type configureCombustion func(ctx *image.Context) error

type Builder struct {
	imageConfig         *image.ImageConfig
	context             *image.Context
	configureCombustion configureCombustion
}

func New(imageConfig *image.ImageConfig, ctx *image.Context, configureCombustionFunc configureCombustion) *Builder {
	return &Builder{
		imageConfig:         imageConfig,
		context:             ctx,
		configureCombustion: configureCombustionFunc,
	}
}

func (b *Builder) Build() error {
	if err := b.configureCombustion(b.context); err != nil {
		return fmt.Errorf("configuring combustion: %w", err)
	}

	switch b.imageConfig.Image.ImageType {
	case image.ImageTypeISO:
		return b.buildIsoImage()
	case image.ImageTypeRAW:
		return b.buildRawImage()
	default:
		return fmt.Errorf("invalid imageType value specified, must be either \"%s\" or \"%s\"",
			image.ImageTypeISO, image.ImageTypeRAW)
	}
}

func (b *Builder) generateBuildDirFilename(filename string) string {
	return filepath.Join(b.context.BuildDir, filename)
}

func (b *Builder) generateOutputImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, b.imageConfig.Image.OutputImageName)
	return filename
}

func (b *Builder) generateBaseImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, "images", b.imageConfig.Image.BaseImage)
	return filename
}
