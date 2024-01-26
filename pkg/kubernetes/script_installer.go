package kubernetes

import (
	"fmt"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

type ScriptInstaller struct{}

func (i ScriptInstaller) InstallScript(distribution, sourcePath, destinationPath string) error {
	const k8sInstallerScript = "%s_installer.sh"
	installer := fmt.Sprintf(k8sInstallerScript, distribution)

	sourcePath = filepath.Join(sourcePath, installer)
	destinationPath = filepath.Join(destinationPath, installer)

	return fileio.CopyFile(sourcePath, destinationPath, fileio.ExecutablePerms)
}
