package network

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

type ConfiguratorInstaller struct{}

func (ConfiguratorInstaller) InstallConfigurator(imageName, sourcePath, installPath string) error {
	const (
		amdArch = "x86_64"
		armArch = "aarch64"
	)

	var arch string

	switch {
	case strings.Contains(imageName, amdArch):
		arch = amdArch
	case strings.Contains(imageName, armArch):
		arch = armArch
	default:
		return fmt.Errorf("failed to determine arch of image %s", imageName)
	}

	const nmcExecutable = "nmc-%s"
	sourcePath = filepath.Join(sourcePath, fmt.Sprintf(nmcExecutable, arch))

	if err := fileio.CopyFile(sourcePath, installPath, fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}
