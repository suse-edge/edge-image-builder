package build

import (
	"fmt"
	"os"
	"path/filepath"
)

func (b *Builder) getRPMFileNames() ([]string, error) {
	var rpmFileNames []string
	rpmSourceDir := filepath.Join(b.buildConfig.ImageConfigDir, "rpms")

	rpms, err := os.ReadDir(rpmSourceDir)
	if err != nil {
		return nil, fmt.Errorf("reading rpm source dir: %w", err)
	}

	for _, rpmFile := range rpms {
		rpmFileNames = append(rpmFileNames, rpmFile.Name())
	}

	if len(rpmFileNames) == 0 {
		return nil, fmt.Errorf("no rpms found")
	} else if len(rpmFileNames) == 1 && rpmFileNames[0] == ".gitkeep" {
		return nil, fmt.Errorf("no rpms found")
	}

	return rpmFileNames, nil
}

func (b *Builder) copyRPMs() error {
	rpmFileNames, err := b.getRPMFileNames()
	if err != nil {
		return fmt.Errorf("getting rpm file names: %w", err)
	}

	rpmSourceDir := filepath.Join(b.buildConfig.ImageConfigDir, "rpms")

	for _, rpm := range rpmFileNames {
		sourcePath := filepath.Join(rpmSourceDir, rpm)
		destPath := filepath.Join(b.combustionDir, rpm)

		err = b.copyFile(sourcePath, destPath)
		if err != nil {
			return fmt.Errorf("looping through rpms to copy: %w", err)
		}
	}

	return nil
}
