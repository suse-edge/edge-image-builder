package build

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	isoExtractDir        = "iso-extract"
	rawExtractDir        = "raw-extract"
	extractIsoScriptName = "iso-extract.sh"
	extractIsoLogFile    = "iso-extract.log"
	rebuildIsoScriptName = "iso-build.sh"
	rebuildIsoLogFile    = "iso-build.log"
)

//go:embed templates/extract-iso.sh.tpl
var extractIsoTemplate string

//go:embed templates/rebuild-iso.sh.tpl
var rebuildIsoTemplate string

func (b *Builder) buildIsoImage() error {
	if err := b.deleteExistingOutputImage(); err != nil {
		return fmt.Errorf("deleting existing ISO image: %w", err)
	}

	if err := b.extractIso(); err != nil {
		return fmt.Errorf("extracting the ISO image: %w", err)
	}

	extractedRawImage, err := b.findExtractedRawImage()
	if err != nil {
		return fmt.Errorf("unable to find extracted raw image: %w", err)
	}
	if err = b.modifyRawImage(extractedRawImage, false, false); err != nil {
		return fmt.Errorf("modifying the raw image inside of the ISO: %w", err)
	}

	if err := b.rebuildIso(); err != nil {
		return fmt.Errorf("building the ISO image: %w", err)
	}

	return nil
}

func (b *Builder) extractIso() error {
	if err := b.writeIsoScript(extractIsoTemplate, extractIsoScriptName); err != nil {
		return fmt.Errorf("creating the ISO extraction script: %w", err)
	}

	cmd, extractLog, err := b.createIsoCommand(extractIsoLogFile, extractIsoScriptName)
	if err != nil {
		return fmt.Errorf("preparing to extract the contents of the ISO: %w", err)
	}
	defer func() {
		if err = extractLog.Close(); err != nil {
			zap.S().Warnf("failed to close ISO extraction log file properly: %s", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("extracting the contents of the ISO: %w", err)
	}

	return nil
}

func (b *Builder) rebuildIso() error {
	if err := b.writeIsoScript(rebuildIsoTemplate, rebuildIsoScriptName); err != nil {
		return fmt.Errorf("creating the ISO rebuild script: %w", err)
	}

	cmd, rebuildLog, err := b.createIsoCommand(rebuildIsoLogFile, rebuildIsoScriptName)
	if err != nil {
		return fmt.Errorf("preparing to build the new ISO: %w", err)
	}
	defer func() {
		if err = rebuildLog.Close(); err != nil {
			zap.S().Warnf("failed to close ISO rebuild log file properly: %s", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("building the new ISO: %w", err)
	}

	return nil
}

func (b *Builder) writeIsoScript(templateContents, outputFilename string) error {
	scriptName := filepath.Join(b.context.BuildDir, outputFilename)
	isoExtractPath := filepath.Join(b.context.BuildDir, isoExtractDir)
	rawExtractPath := filepath.Join(b.context.BuildDir, rawExtractDir)
	arguments := struct {
		IsoExtractDir       string
		RawExtractDir       string
		IsoSource           string
		OutputImageFilename string
		CombustionDir       string
		ArtefactsDir        string
		InstallDevice       string
	}{
		IsoExtractDir:       isoExtractPath,
		RawExtractDir:       rawExtractPath,
		IsoSource:           b.generateBaseImageFilename(),
		OutputImageFilename: b.generateOutputImageFilename(),
		CombustionDir:       b.context.CombustionDir,
		ArtefactsDir:        b.context.ArtefactsDir,
		InstallDevice:       b.context.ImageDefinition.OperatingSystem.IsoConfiguration.InstallDevice,
	}

	contents, err := template.Parse("iso-script", templateContents, arguments)
	if err != nil {
		return fmt.Errorf("applying the ISO script template: %w", err)
	}

	if err = os.WriteFile(scriptName, []byte(contents), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing ISO extraction script %s: %w", outputFilename, err)
	}

	return nil
}

func (b *Builder) createIsoCommand(logFilename, scriptName string) (*exec.Cmd, *os.File, error) {
	fullLogFilename := filepath.Join(b.context.BuildDir, logFilename)
	logFile, err := os.Create(fullLogFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening ISO log file %s: %w", logFilename, err)
	}

	scriptFilename := filepath.Join(b.context.BuildDir, scriptName)
	cmd := exec.Command(scriptFilename)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd, logFile, nil
}

func (b *Builder) findExtractedRawImage() (string, error) {
	var foundFile string
	f := func(s string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if strings.HasSuffix(info.Name(), ".raw") {
			foundFile = info.Name()
		}
		return nil
	}

	rawExtractPath := filepath.Join(b.context.BuildDir, rawExtractDir)
	if err := filepath.Walk(rawExtractPath, f); err != nil {
		return "", fmt.Errorf("traversing raw extract directory %s: %w", rawExtractPath, err)
	}

	if foundFile == "" {
		return "", fmt.Errorf("unable to find a raw image in: %s", rawExtractDir)
	}

	return filepath.Join(rawExtractPath, foundFile), nil
}
