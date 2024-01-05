package network

import (
	"fmt"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type ConfiguratorInstaller struct{}

func (ConfiguratorInstaller) InstallConfigurator(arch image.Arch, sourcePath, installPath string) error {
	const nmcExecutable = "nmc-%s"
	sourcePath = filepath.Join(sourcePath, fmt.Sprintf(nmcExecutable, arch))

	if err := fileio.CopyFile(sourcePath, installPath, fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}
