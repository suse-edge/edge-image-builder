package build

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

const (
	modifyRPMScriptName = "10_rpm_install.sh"
)

//go:embed scripts/rpms/10-rpm-install.sh.tpl
var modifyRPMScript string

func (b *Builder) processRPMs() error {
	RPMSourceDir, err := b.generateRPMPath()
	if err != nil {
		return fmt.Errorf("generating RPM path: %w", err)
	}
	// Only proceed with processing the RPMs if the directory exists
	if RPMSourceDir == "" {
		return nil
	}

	RPMFileNames, err := getRPMFileNames(RPMSourceDir)
	if err != nil {
		return fmt.Errorf("getting RPM file names: %w", err)
	}

	err = copyRPMs(RPMSourceDir, b.combustionDir, RPMFileNames)
	if err != nil {
		return fmt.Errorf("copying RPMs over: %w", err)
	}

	err = b.writeRPMScript(RPMFileNames)
	if err != nil {
		return fmt.Errorf("writing the RPM install script %s: %w", modifyRPMScriptName, err)
	}

	return nil
}

func getRPMFileNames(RPMSourceDir string) ([]string, error) {
	var RPMFileNames []string

	RPMs, err := os.ReadDir(RPMSourceDir)
	if err != nil {
		return nil, fmt.Errorf("reading RPM source dir: %w", err)
	}

	for _, RPMFile := range RPMs {
		if filepath.Ext(RPMFile.Name()) == ".rpm" {
			RPMFileNames = append(RPMFileNames, RPMFile.Name())
		}
	}

	if len(RPMFileNames) == 0 {
		return nil, fmt.Errorf("no RPMs found")
	}

	return RPMFileNames, nil
}

func copyRPMs(RPMSourceDir string, RPMDestDir string, RPMFileNames []string) error {
	for _, RPM := range RPMFileNames {
		sourcePath := filepath.Join(RPMSourceDir, RPM)
		destPath := filepath.Join(RPMDestDir, RPM)

		err := fileio.CopyFile(sourcePath, destPath)
		if err != nil {
			return fmt.Errorf("copying file %s: %w", sourcePath, err)
		}
	}

	return nil
}

func (b *Builder) writeRPMScript(RPMFileNames []string) error {
	values := struct {
		RPMs string
	}{
		RPMs: strings.Join(RPMFileNames, " "),
	}

	writtenFilename, err := b.writeCombustionFile(modifyRPMScriptName, modifyRPMScript, &values)
	if err != nil {
		return fmt.Errorf("writing RPM script %s: %w", modifyRPMScriptName, err)
	}
	err = os.Chmod(writtenFilename, modifyScriptMode)
	if err != nil {
		return fmt.Errorf("adjusting permissions: %w", err)
	}

	b.registerCombustionScript(modifyRPMScriptName)

	return nil
}

func (b *Builder) generateRPMPath() (string, error) {
	RPMSourceDir := filepath.Join(b.buildConfig.ImageConfigDir, "rpms")
	_, err := os.Stat(RPMSourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("checking for RPM directory at %s: %w", RPMSourceDir, err)
	}

	return RPMSourceDir, nil
}
