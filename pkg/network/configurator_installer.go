package network

import (
	"fmt"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

type ConfiguratorInstaller struct{}

func (ConfiguratorInstaller) InstallConfigurator(sourcePath, installPath string) error {
	if err := fileio.CopyFile(sourcePath, installPath, fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}
