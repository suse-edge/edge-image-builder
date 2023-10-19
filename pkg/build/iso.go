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
	xorrisoArgs := b.generateXorrisoArgs()
	err := b.runXorriso(xorrisoArgs)
	return err
}

func (b *Builder) generateXorrisoArgs() string {
	indevPath := filepath.Join(b.buildConfig.ImageConfigDir, "images", b.imageConfig.Image.BaseImage)
	outdevPath := filepath.Join(b.buildConfig.ImageConfigDir, b.imageConfig.Image.OutputImageName)
	mapDir := b.combustionDir

	args := fmt.Sprintf(xorrisoArgsBase, indevPath, outdevPath, mapDir)
	return args
}

func (b *Builder) runXorriso(args string) error {
	splitArgs := strings.Split(args, " ")
	cmd := exec.Command(xorrisoExec, splitArgs...)

	logFilename := filepath.Join(b.eibBuildDir, xorrisoLogFile)
	xorrisoLog, err := os.Create(logFilename)
	if err != nil {
		return fmt.Errorf("opening ISO build logfile: %w", err)
	}
	defer xorrisoLog.Close()
	cmd.Stdout = xorrisoLog
	cmd.Stderr = xorrisoLog

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running xorriso: %w", err)
	}

	zap.L().Sugar().Debugf("ISO log file created: %s", logFilename)

	return nil
}
