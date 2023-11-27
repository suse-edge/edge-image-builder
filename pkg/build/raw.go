package build

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	copyExec         = "/bin/cp"
	modifyScriptName = "modify-raw-image.sh"
)

//go:embed scripts/modify-raw-image.sh.tpl
var modifyRawImageTemplate string

func (b *Builder) buildRawImage() error {
	cmd := b.createRawImageCopyCommand()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("copying the base image %s to the output image location %s: %w",
			b.context.ImageDefinition.Image.BaseImage, b.generateOutputImageFilename(), err)
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
	// There is no need to check the returned results from this call. If there is no configuration,
	// it will be an empty string, which is safe to pass into the template.
	grubConfiguration, err := b.generateGRUBGuestfishCommands()
	if err != nil {
		return fmt.Errorf("generating the GRUB configuration commands: %w", err)
	}

	// Assemble the template values
	values := struct {
		OutputImage   string
		CombustionDir string
		ConfigureGRUB string
	}{
		OutputImage:   b.generateOutputImageFilename(),
		CombustionDir: b.context.CombustionDir,
		ConfigureGRUB: grubConfiguration,
	}

	data, err := template.Parse(modifyScriptName, modifyRawImageTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", modifyScriptName, err)
	}

	filename := b.generateBuildDirFilename(modifyScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing modification script %s: %w", modifyScriptName, err)
	}

	return nil
}

func (b *Builder) createModifyCommand() *exec.Cmd {
	scriptPath := filepath.Join(b.context.BuildDir, modifyScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}
