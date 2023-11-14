package build

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"strings"
)

const (
	modifyRPMScriptName = "10_rpm_install.sh"
)

//go:embed scripts/rpms/10_rpm_install.sh.tpl
var modifyRPMScript string

func (b *Builder) processRPMs() error {
	rpmSourceDir, err := b.generateRPMPath()
	if err != nil {
		return fmt.Errorf("generating rpm path: %w", err)
	}
	rpmDestDir := b.combustionDir

	rpmFileNames, err := getRPMFileNames(rpmSourceDir)
	if err != nil {
		return fmt.Errorf("getting rpm file names: %w", err)
	}

	err = copyRPMs(rpmSourceDir, rpmDestDir, rpmFileNames)
	if err != nil {
		return fmt.Errorf("copying RPMs over: %w", err)
	}

	err = b.writeRPMScript(rpmFileNames)
	if err != nil {
		return fmt.Errorf("writing the rpm install script: %w", err)
	}

	return nil
}

func getRPMFileNames(rpmSourceDir string) ([]string, error) {
	var rpmFileNames []string

	rpms, err := os.ReadDir(rpmSourceDir)
	if err != nil {
		return nil, fmt.Errorf("reading rpm source dir: %w", err)
	}

	for _, rpmFile := range rpms {
		if filepath.Ext(rpmFile.Name()) == ".rpm" {
			rpmFileNames = append(rpmFileNames, rpmFile.Name())
		}
	}

	if len(rpmFileNames) == 0 {
		return nil, fmt.Errorf("no rpms found")
	}

	return rpmFileNames, nil
}

func copyRPMs(rpmSourceDir string, rpmDestDir string, rpmFileNames []string) error {
	for _, rpm := range rpmFileNames {
		sourcePath := filepath.Join(rpmSourceDir, rpm)
		destPath := filepath.Join(rpmDestDir, rpm)

		err := fileio.CopyFile(sourcePath, destPath)
		if err != nil {
			return fmt.Errorf("copying file %s: %w", sourcePath, err)
		}
	}

	return nil
}

func (b *Builder) writeRPMScript(rpmFileNamesArray []string) error {
	rpmFileNamesString := strings.Join(rpmFileNamesArray, " ")
	values := struct {
		RPMs string
	}{
		RPMs: rpmFileNamesString,
	}

	writtenFilename, err := b.writeCombustionFile(modifyRPMScriptName, modifyRPMScript, &values)
	if err != nil {
		return fmt.Errorf("writing rpm script %s: %w", modifyRPMScriptName, err)
	}
	err = os.Chmod(writtenFilename, modifyScriptMode)
	if err != nil {
		return fmt.Errorf("changing permissions on the rpm script %s: %w", modifyRPMScriptName, err)
	}

	b.registerCombustionScript(modifyRPMScriptName)

	return nil
}

func (b *Builder) generateRPMPath() (string, error) {
	rpmSourceDir := filepath.Join(b.buildConfig.ImageConfigDir, "rpms")
	// Only proceed with processing the RPMs if the directory exists
	_, err := os.Stat(rpmSourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("checking for rpm directory at %s: %w", rpmSourceDir, err)
	}

	return rpmSourceDir, nil
}
