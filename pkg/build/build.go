package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"github.com/suse-edge/edge-image-builder/pkg/config"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

type Builder struct {
	imageConfig *config.ImageConfig
	context     *Context
}

func New(imageConfig *config.ImageConfig, context *Context) *Builder {
	return &Builder{
		imageConfig: imageConfig,
		context:     context,
	}
}

func (b *Builder) Build() error {
	var combustionScripts []string

	messageScript, err := b.configureMessage()
	if err != nil {
		return fmt.Errorf("configuring the welcome message: %w", err)
	}

	combustionScripts = append(combustionScripts, messageScript)

	customScripts, err := b.configureCustomScripts()
	if err != nil {
		return fmt.Errorf("configuring custom scripts: %w", err)
	}

	combustionScripts = append(combustionScripts, customScripts...)

	userScript, err := b.configureUsers()
	if err != nil {
		return fmt.Errorf("configuring users: %w", err)
	}

	if userScript != "" {
		combustionScripts = append(combustionScripts, userScript)
	}

	rpmScript, err := b.processRPMs()
	if err != nil {
		return fmt.Errorf("processing RPMs: %w", err)
	}

	if rpmScript != "" {
		combustionScripts = append(combustionScripts, rpmScript)
	}

	script, err := combustion.GenerateScript(combustionScripts)
	if err != nil {
		return fmt.Errorf("generating combustion script: %w", err)
	}

	if err = os.WriteFile("script", []byte(script), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing combustion script: %w", err)
	}

	switch b.imageConfig.Image.ImageType {
	case config.ImageTypeISO:
		return b.buildIsoImage()
	case config.ImageTypeRAW:
		return b.buildRawImage()
	default:
		return fmt.Errorf("invalid imageType value specified, must be either \"%s\" or \"%s\"",
			config.ImageTypeISO, config.ImageTypeRAW)
	}
}

func (b *Builder) generateBuildDirFilename(filename string) string {
	return filepath.Join(b.context.BuildDir, filename)
}

func (b *Builder) generateCombustionDirFilename(filename string) string {
	return filepath.Join(b.context.CombustionDir, filename)
}

func (b *Builder) generateOutputImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, b.imageConfig.Image.OutputImageName)
	return filename
}

func (b *Builder) generateBaseImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, "images", b.imageConfig.Image.BaseImage)
	return filename
}
