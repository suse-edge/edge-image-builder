package build

import (
	"fmt"
	"os"
	"path/filepath"
)

func (b *Builder) getRPMFileNames(rpmSourceDir string) ([]string, error) {
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

func (b *Builder) copyRPMs() error {
	rpmSourceDir := filepath.Join(b.buildConfig.ImageConfigDir, "rpms")
	// Only proceed with copying the RPMs if the directory exists
	_, err := os.Stat(rpmSourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return fmt.Errorf("checking for rpm directory at %s: %w", rpmSourceDir, err)
		}
	}
	rpmDestDir := b.combustionDir

	rpmFileNames, err := b.getRPMFileNames(rpmSourceDir)
	if err != nil {
		return fmt.Errorf("getting rpm file names: %w", err)
	}

	for _, rpm := range rpmFileNames {
		sourcePath := filepath.Join(rpmSourceDir, rpm)
		destPath := filepath.Join(rpmDestDir, rpm)

		err = copyFile(sourcePath, destPath)
		if err != nil {
			return fmt.Errorf("copying file %s: %w", sourcePath, err)
		}
	}

	return nil
}
