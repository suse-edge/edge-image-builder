package kubernetes

import (
	"fmt"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

type ScriptInstaller struct{}

func (i ScriptInstaller) InstallScript(distribution, sourcePath, destinationPath string) error {
	const rke2InstallerScript = "%s_installer.sh"
	installer := fmt.Sprintf(rke2InstallerScript, distribution)

	sourcePath = filepath.Join(sourcePath, installer)
	destinationPath = filepath.Join(destinationPath, installer)

	return fileio.CopyFile(sourcePath, destinationPath, fileio.ExecutablePerms)
}
