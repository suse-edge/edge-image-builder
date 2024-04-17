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
	copyExec                = "/bin/cp"
	modifyScriptName        = "modify-raw-image.sh"
	rawBuildLogFile         = "raw-build.log"
	availableRawDiskSpaceMB = 150
)

//go:embed templates/modify-raw-image.sh.tpl
var modifyRawImageTemplate string

func (b *Builder) buildRawImage() error {
	requiredSpace, err := b.calculateMinimumRequiredSpace()
	if err != nil {
		return fmt.Errorf("calculating minimum required space: %w", err)
	}

	imageSize, err := b.retrieveImageSize()
	if err != nil {
		return fmt.Errorf("retrieving RAW base image size: %w", err)
	}

	diskSize := b.context.ImageDefinition.OperatingSystem.RawConfiguration.DiskSize.ToMB()
	if diskSize <= imageSize+requiredSpace && requiredSpace >= availableRawDiskSpaceMB {
		zap.S().Warnf("Insufficient available disk space. The build artifacts require an expansion of the base image by least %d MB. "+
			"Please specify an appropriate disk size taking into consideration that some of the artifacts may be compressed.",
			requiredSpace)
		return fmt.Errorf("insufficient available disk space on the RAW image")
	}

	if err = b.deleteExistingOutputImage(); err != nil {
		return fmt.Errorf("deleting existing RAW image: %w", err)
	}

	cmd := b.createRawImageCopyCommand()
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("copying the base image %s to the output image location %s: %w",
			b.context.ImageDefinition.Image.BaseImage, b.generateOutputImageFilename(), err)
	}

	return b.modifyRawImage(b.generateOutputImageFilename(), true, true)
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
		ArtefactsDir        string
		ConfigureGRUB       string
		ConfigureCombustion bool
		RenameFilesystem    bool
		DiskSize            string
	}{
		ImagePath:           imageFilename,
		CombustionDir:       b.context.CombustionDir,
		ArtefactsDir:        b.context.ArtefactsDir,
		ConfigureGRUB:       grubConfiguration,
		ConfigureCombustion: includeCombustion,
		RenameFilesystem:    renameFilesystem,
		DiskSize:            string(b.context.ImageDefinition.OperatingSystem.RawConfiguration.DiskSize),
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

// Retrieve the size of the base image in MB.
func (b *Builder) retrieveImageSize() (int64, error) {
	imageFile, err := os.Stat(b.generateBaseImageFilename())
	if err != nil {
		return 0, fmt.Errorf("reading base image file info: %w", err)
	}

	return imageFile.Size() / (1024 * 1024), nil
}

// Calculate the size (in MB) of the artefacts which will be copied in the built image.
func (b *Builder) calculateMinimumRequiredSpace() (int64, error) {
	var requiredSpace int64

	size, err := dirSize(b.context.CombustionDir)
	if err != nil {
		return 0, fmt.Errorf("calculating combustion directory size: %w", err)
	}
	requiredSpace += size

	size, err = dirSize(b.context.ArtefactsDir)
	if err != nil {
		return 0, fmt.Errorf("calculating artefacts directory size: %w", err)
	}
	requiredSpace += size

	return requiredSpace, nil
}

// Traverse a directory and all of its subdirectories
// returning the total size of their contents in MB.
func dirSize(path string) (int64, error) {
	var size int64

	calculateSize := func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	}

	return size / (1024 * 1024), filepath.Walk(path, calculateSize)
}
