package build

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	copyExec = "/bin/cp"
	modifyScriptName = "modify-raw-image.sh"
)

//go:embed scripts/modify-raw-image.sh
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
	imageToModify := b.generateOutputImageFilename()
	contents := fmt.Sprintf(modifyRawImageScript, imageToModify, b.combustionDir)

	scriptPath := filepath.Join(b.buildConfig.BuildDir, modifyScriptName)
	err := os.WriteFile(scriptPath, []byte(contents), os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing file %s: %w", scriptPath, err)
	}

	return nil
}

func (b *Builder) createModifyCommand() *exec.Cmd {
	scriptPath := filepath.Join(b.buildConfig.BuildDir, modifyScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}