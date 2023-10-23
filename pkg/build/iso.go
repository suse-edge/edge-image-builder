package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	xorrisoArgsBase = "-indev %s -outdev %s -map %s /combustion -boot_image any replay -changes_pending yes"
	xorrisoExec     = "/usr/bin/xorriso"
	xorrisoLogFile  = "iso-build-%s.log"
)

func (b *Builder) buildIsoImage() error {
	err := b.deleteExistingOutputIso()
	if err != nil {
		return fmt.Errorf("deleting existing ISO image: %w", err)
	}

	cmd, logfile, err := b.createXorrisoCommand()
	if err != nil {
		return fmt.Errorf("configuring the ISO build command: %w", err)
	}
	defer logfile.Close()

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running xorriso: %w", err)
	}

	return err
}

func (b *Builder) deleteExistingOutputIso() error {
	outputFilename := b.generateOutputIsoFilename()
	err := os.Remove(outputFilename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error deleting file %s: %w", outputFilename, err)
	}
	return nil
}

func (b *Builder) createXorrisoCommand() (*exec.Cmd, *os.File, error) {
	logFilename := b.generateIsoLogFilename()
	xorrisoLog, err := os.Create(logFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("opening ISO build logfile: %w", err)
	}
	zap.L().Sugar().Debugf("ISO log file created: %s", logFilename)

	args := b.generateXorrisoArgs()
	cmd := exec.Command(xorrisoExec, args...)
	cmd.Stdout = xorrisoLog
	cmd.Stderr = xorrisoLog

	return cmd, xorrisoLog, nil
}

func (b *Builder) generateXorrisoArgs() []string {
	indevPath := filepath.Join(b.buildConfig.ImageConfigDir, "images", b.imageConfig.Image.BaseImage)
	outdevPath := filepath.Join(b.buildConfig.ImageConfigDir, b.imageConfig.Image.OutputImageName)
	mapDir := b.combustionDir

	args := fmt.Sprintf(xorrisoArgsBase, indevPath, outdevPath, mapDir)
	splitArgs := strings.Split(args, " ")
	return splitArgs
}

func (b *Builder) generateIsoLogFilename() string {
	timestamp := time.Now().Format("Jan02_15-04-05")
	filename := fmt.Sprintf(xorrisoLogFile, timestamp)
	logFilename := filepath.Join(b.eibBuildDir, filename)
	return logFilename
}

func (b *Builder) generateOutputIsoFilename() string {
	filename := filepath.Join(b.buildConfig.ImageConfigDir, b.imageConfig.Image.OutputImageName)
	return filename
}
