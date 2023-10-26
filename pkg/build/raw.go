package build

import (
	_ "embed"
	"fmt"
	"os/exec"
	"path/filepath"
)

const (
	copyExec         = "/bin/cp"
	modifyScriptName = "modify-raw-image.sh"
)

//go:embed scripts/modify-raw-image.sh.tpl
var modifyRawImageScript string

func (b *Builder) buildRawImage() error {
	cmd := b.createRawImageCopyCommand()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("copying the base image %s to the output image location %s: %w",
			b.imageConfig.Image.BaseImage, b.generateOutputImageFilename(), err)
	}

	err = b.writeModifyScript()
	if err != nil {
		return fmt.Errorf("writing the image modification script: %w", err)
	}

	cmd = b.createModifyCommand()
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("running the image modification script: %w", err)
	}

	return nil
}

func (b *Builder) createRawImageCopyCommand() *exec.Cmd {
	baseImagePath := b.generateBaseImageFilename()
	outputImagePath := b.generateOutputImageFilename()

	cmd := exec.Command(copyExec, baseImagePath, outputImagePath)
	return cmd
}

func (b *Builder) writeModifyScript() error {
	values := struct {
		OutputImage   string
		CombustionDir string
	}{
		OutputImage:   b.generateOutputImageFilename(),
		CombustionDir: b.combustionDir,
	}

	err := b.writeBuildDirFile(modifyRawImageScript, &values, modifyScriptName)
	if err != nil {
		return fmt.Errorf("writing modification script %s: %w", modifyScriptName, err)
	}

	return nil
}

func (b *Builder) createModifyCommand() *exec.Cmd {
	scriptPath := filepath.Join(b.buildConfig.BuildDir, modifyScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}
