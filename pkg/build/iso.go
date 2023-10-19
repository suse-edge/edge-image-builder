package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

const (
	xorrisoArgsBase = "-indev %s -outdev %s -map %s /combustion -boot_image any replay -changes_pending yes"
	xorrisoExec     = "/usr/bin/xorriso"
	xorrisoLogFile  = "iso-build.log"
)

func (b *Builder) buildIsoImage() error {
	cmd, err := b.createXorrisoCommand()

	if err != nil {
		return err
	}

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running xorriso: %w", err)
	} else {
		zap.L().Sugar().Debugf("ISO log file created: %s", b.generateIsoLogFilename())
	}

	return err
}

func (b *Builder) createXorrisoCommand() (*exec.Cmd, error) {
	args := b.generateXorrisoArgs()
	cmd := exec.Command(xorrisoExec, args...)

	logFilename := b.generateIsoLogFilename()
	xorrisoLog, err := os.Create(logFilename)
	if err != nil {
		return nil, fmt.Errorf("opening ISO build logfile: %w", err)
	}
	defer xorrisoLog.Close()
	cmd.Stdout = xorrisoLog
	cmd.Stderr = xorrisoLog

	return cmd, nil
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
	logFilename := filepath.Join(b.eibBuildDir, xorrisoLogFile)
	return logFilename
}