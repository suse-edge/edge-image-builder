package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (b *Builder) getRPMFileNames() error {
	b.rpmSourceDir = filepath.Join(b.buildConfig.ImageConfigDir, "rpms")

	rpms, err := os.ReadDir(b.rpmSourceDir)
	if err != nil {
		return fmt.Errorf("reading rpm source dir: %w", err)
	}

	for _, rpmFile := range rpms {
		b.rpmFileNames = append(b.rpmFileNames, rpmFile.Name())
	}

	return nil
}

func (b *Builder) copyRPMs() error {
	err := b.getRPMFileNames()
	if err != nil {
		return fmt.Errorf("getting rpm file names: %w", err)
	}

	for _, rpm := range b.rpmFileNames {
		sourcePath := filepath.Join(b.rpmSourceDir, rpm)
		destPath := filepath.Join(b.combustionDir, rpm)

		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			return fmt.Errorf("opening rpm source path: %w", err)
		}
		defer sourceFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("opening rpm dest path: %w", err)
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, sourceFile)
		if err != nil {
			return fmt.Errorf("copying rpm: %w", err)
		}
	}

	return nil
}
