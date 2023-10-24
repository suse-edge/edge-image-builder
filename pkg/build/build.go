package build

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/config"
)

const (
	embeddedScriptsBaseDir = "scripts"
)

//go:embed scripts/script_base.sh
var combustionScriptBaseCode string

type Builder struct {
	imageConfig *config.ImageConfig
	buildConfig *config.BuildConfig

	eibBuildDir       string
	combustionDir     string
	combustionScripts []string
}

func New(imageConfig *config.ImageConfig, buildConfig *config.BuildConfig) *Builder {
	return &Builder{
		imageConfig: imageConfig,
		buildConfig: buildConfig,
	}
}

func (b *Builder) Build() error {
	err := b.prepareBuildDir()
	if err != nil {
		return fmt.Errorf("preparing the build directory: %w", err)
	}

	err = b.configureMessage()
	if err != nil {
		return fmt.Errorf("configuring the welcome message: %w", err)
	}

	err = b.generateCombustionScript()
	if err != nil {
		return fmt.Errorf("generating combustion script: %w", err)
	}

	err = b.buildIsoImage()
	if err != nil {
		return fmt.Errorf("error building modified ISO: %w", err)
	}

	// Temporarily disabling; will add a flag to control this next
	//err = b.cleanUpBuildDir()
	//if err != nil {
	//	return fmt.Errorf("cleaning up the build directory: %w", err)
	//}

	return nil
}

func (b *Builder) prepareBuildDir() error {
	// Combustion works by creating a volume with a subdirectory named "combustion"
	// and a file named "script". This function builds out that structure and updates
	// the Builder so that the other functions can populate it as necessary.

	if b.buildConfig.BuildDir == "" {
		tmpDir, err := os.MkdirTemp("", "eib-")
		if err != nil {
			return fmt.Errorf("creating a temporary build directory: %w", err)
		}
		b.eibBuildDir = tmpDir
	} else {
		b.eibBuildDir = b.buildConfig.BuildDir
	}
	b.combustionDir = filepath.Join(b.eibBuildDir, "combustion")

	err := os.MkdirAll(b.combustionDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating the build directory structure: %w", err)
	}

	return nil
}

func (b *Builder) cleanUpBuildDir() error {
	err := os.RemoveAll(b.eibBuildDir)
	if err != nil {
		return fmt.Errorf("deleting build directory: %w", err)
	}
	return nil
}

func (b *Builder) generateCombustionScript() error {
	// The file must be located at "combustion/script"
	scriptFilename := filepath.Join(b.combustionDir, "script")
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
	for _, filename := range b.combustionScripts {
		_, err = fmt.Fprintln(scriptFile, "./"+filename)
		if err != nil {
			return fmt.Errorf("modifying the combustion script to add %s: %w", filename, err)
		}
	}

	return nil
}

func (b *Builder) writeCombustionFile(contents string, filename string) error {
	destFilename := filepath.Join(b.combustionDir, filename)
	err := os.WriteFile(destFilename, []byte(contents), os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing file %s: %w", destFilename, err)
	}

	return nil
}

func (b *Builder) copyCombustionFile(scriptSubDir string, scriptName string) error {
	sourcePath := filepath.Join(embeddedScriptsBaseDir, scriptSubDir, scriptName)
	src, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	destFilename := filepath.Join(b.combustionDir, filepath.Base(sourcePath))
	err = os.WriteFile(destFilename, src, os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func (b *Builder) registerCombustionScript(scriptName string) {
	// Keep a running list of all added combustion scripts. When we add the combustion
	// "script" file (the one Combustion itself looks at), we'll concatenate calls to
	// each of these to that script.

	b.combustionScripts = append(b.combustionScripts, scriptName)
}

func (b *Builder) generateOutputImageFilename() string {
	filename := filepath.Join(b.buildConfig.ImageConfigDir, b.imageConfig.Image.OutputImageName)
	return filename
}
