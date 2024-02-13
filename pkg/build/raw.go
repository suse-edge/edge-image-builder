package build

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	copyExec         = "/bin/cp"
	modifyScriptName = "modify-raw-image.sh"
	rawBuildLogFile  = "raw-build.log"
)

//go:embed templates/modify-raw-image.sh.tpl
var modifyRawImageTemplate string

func (b *Builder) buildRawImage() error {
	cmd := b.createRawImageCopyCommand()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copying the base image %s to the output image location %s: %w",
			b.context.ImageDefinition.Image.BaseImage, b.generateOutputImageFilename(), err)
	}

	if err := b.modifyRawImage(b.generateOutputImageFilename(), true, true); err != nil {
		// modifyRawImage will wrap the error, simply return it here
		return err
	}

	return nil
}

func (b *Builder) modifyRawImage(imagePath string, includeCombustion, renameFilesystem bool) error {
	if err := b.writeModifyScript(imagePath, includeCombustion, renameFilesystem); err != nil {
		return fmt.Errorf("writing the image modification script: %w", err)
	}

	logFilename := filepath.Join(b.context.BuildDir, rawBuildLogFile)
	logFile, err := os.Create(logFilename)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}

	defer func() {
		if err = logFile.Close(); err != nil {
			zap.S().Warnf("Failed to close raw build log file properly: %s", err)
		}
	}()

	cmd := b.createModifyCommand(logFile)
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

func (b *Builder) writeModifyScript(imageFilename string, includeCombustion, renameFilesystem bool) error {
	// There is no need to check the returned results from this call. If there is no configuration,
	// it will be an empty string, which is safe to pass into the template.
	grubConfiguration, err := b.generateGRUBGuestfishCommands()
	if err != nil {
		return fmt.Errorf("generating the GRUB configuration commands: %w", err)
	}

	// Assemble the template values
	values := struct {
		ImagePath           string
		CombustionDir       string
		ConfigureGRUB       string
		ConfigureCombustion bool
		RenameFilesystem    bool
		DiskSize            string
	}{
		ImagePath:           imageFilename,
		CombustionDir:       b.context.CombustionDir,
		ConfigureGRUB:       grubConfiguration,
		ConfigureCombustion: includeCombustion,
		RenameFilesystem:    renameFilesystem,
		DiskSize:            b.context.ImageDefinition.OperatingSystem.RawConfiguration.DiskSize,
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

func (b *Builder) createModifyCommand(writer io.Writer) *exec.Cmd {
	scriptPath := filepath.Join(b.context.BuildDir, modifyScriptName)

	cmd := exec.Command(scriptPath)
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd
}
