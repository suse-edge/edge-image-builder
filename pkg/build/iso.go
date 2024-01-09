package build

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	if err := b.deleteExistingOutputIso(); err != nil {
		return fmt.Errorf("deleting existing ISO image: %w", err)
	}

	if err := b.extractIso(); err != nil {
		return fmt.Errorf("extracting the ISO image: %w", err)
	}

	// TODO: Call into raw code

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
			zap.S().Warn("failed to close ISO extraction log file properly", zap.Error(err))
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
			zap.S().Warn("failed to close ISO rebuild log file properly", zap.Error(err))
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("building the new ISO: %w", err)
	}

	return nil
}

func (b *Builder) deleteExistingOutputIso() error {
	outputFilename := b.generateOutputImageFilename()
	err := os.Remove(outputFilename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error deleting file %s: %w", outputFilename, err)
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
	}{
		IsoExtractDir:       isoExtractPath,
		RawExtractDir:       rawExtractPath,
		IsoSource:           b.generateBaseImageFilename(),
		OutputImageFilename: b.generateOutputImageFilename(),
		CombustionDir:       b.context.CombustionDir,
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
