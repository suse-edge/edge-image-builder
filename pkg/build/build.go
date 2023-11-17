package build

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/suse-edge/edge-image-builder/pkg/config"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

//go:embed scripts/script_base.sh
var combustionScriptBaseCode string

type Builder struct {
	imageConfig *config.ImageConfig
	context     *Context

	combustionScripts []string
}

func New(imageConfig *config.ImageConfig, context *Context) *Builder {
	return &Builder{
		imageConfig: imageConfig,
		context:     context,
	}
}

func (b *Builder) Build() error {
	err := b.configureMessage()
	if err != nil {
		return fmt.Errorf("configuring the welcome message: %w", err)
	}

	err = b.configureScripts()
	if err != nil {
		return fmt.Errorf("configuring custom scripts: %w", err)
	}

	err = b.processRPMs()
	if err != nil {
		return fmt.Errorf("processing RPMs: %w", err)
	}

	err = b.generateCombustionScript()
	if err != nil {
		return fmt.Errorf("generating combustion script: %w", err)
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

func (b *Builder) generateCombustionScript() error {
	// The file must be located at "combustion/script"
	scriptFilename := filepath.Join(b.context.CombustionDir, "script")
	scriptFile, err := os.Create(scriptFilename)
	if err != nil {
		return fmt.Errorf("creating the combustion \"script\" file: %w", err)
	}
	defer scriptFile.Close()

	// Write the script initialization lines
	_, err = fmt.Fprintln(scriptFile, combustionScriptBaseCode)
	if err != nil {
		return fmt.Errorf("writing the combustion \"script\" basefile: %w", err)
	}

	// Add a call to each script that was added to the combustion directory
	// We may need a better way of specifying the order, but for now use alphabetical
	// so we have at least some determinism
	slices.Sort(b.combustionScripts)
	for _, filename := range b.combustionScripts {
		_, err = fmt.Fprintln(scriptFile, "./"+filename)
		if err != nil {
			return fmt.Errorf("modifying the combustion script to add %s: %w", filename, err)
		}
	}

	return nil
}

func (b *Builder) writeBuildDirFile(filename string, contents string, templateData any) (string, error) {
	destFilename := filepath.Join(b.context.BuildDir, filename)
	return destFilename, fileio.WriteFile(destFilename, contents, templateData)
}

func (b *Builder) writeCombustionFile(filename string, contents string, templateData any) (string, error) {
	destFilename := filepath.Join(b.context.CombustionDir, filename)
	return destFilename, fileio.WriteFile(destFilename, contents, templateData)
}

func (b *Builder) registerCombustionScript(scriptName string) {
	// Keep a running list of all added combustion scripts. When we add the combustion
	// "script" file (the one Combustion itself looks at), we'll concatenate calls to
	// each of these to that script.

	b.combustionScripts = append(b.combustionScripts, scriptName)
}

func (b *Builder) generateOutputImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, b.imageConfig.Image.OutputImageName)
	return filename
}

func (b *Builder) generateBaseImageFilename() string {
	filename := filepath.Join(b.context.ImageConfigDir, "images", b.imageConfig.Image.BaseImage)
	return filename
}
