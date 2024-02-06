package build

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

type configureCombustion func(ctx *image.Context) error

type Builder struct {
	context             *image.Context
	configureCombustion configureCombustion
}

func NewBuilder(ctx *image.Context) *Builder {
	return &Builder{
		context:             ctx,
		configureCombustion: combustion.Configure,
	}
}

func (b *Builder) Build() error {
	log.Audit("Generating image customization components...")

	if err := b.configureCombustion(b.context); err != nil {
		log.Audit("Error configuring customization components, check the logs under the build directory for more information.")
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
	filename := filepath.Join(b.context.ImageConfigDir, "base-images", b.context.ImageDefinition.Image.BaseImage)
	return filename
}

func SetupBuildDirectory(rootDir string) (buildDir string, combustionDir string, err error) {
	if rootDir == "" {
		tmpDir, tmpErr := os.MkdirTemp("", "eib-")
		if tmpErr != nil {
			return "", "", fmt.Errorf("creating a temporary root directory: %w", tmpErr)
		}
		rootDir = tmpDir
	}

	timestamp := time.Now().Format("Jan02_15-04-05")
	buildDir = filepath.Join(rootDir, fmt.Sprintf("build-%s", timestamp))
	if err = os.MkdirAll(buildDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating a build directory: %w", err)
	}

	combustionDir = filepath.Join(buildDir, "combustion")
	if err = os.MkdirAll(combustionDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating a combustion directory: %w", err)
	}

	return buildDir, combustionDir, nil
}
