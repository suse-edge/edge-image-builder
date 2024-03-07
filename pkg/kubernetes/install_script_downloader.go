package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/http"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	rke2InstallScriptURL = "https://get.rke2.io"
	k3sInstallScriptURL  = "https://get.k3s.io"
)

type ScriptDownloader struct{}

func (d ScriptDownloader) DownloadInstallScript(distribution, destinationPath string) (string, error) {
	var scriptURL string

	switch distribution {
	case image.KubernetesDistroRKE2:
		scriptURL = rke2InstallScriptURL
	case image.KubernetesDistroK3S:
		scriptURL = k3sInstallScriptURL
	default:
		return "", fmt.Errorf("unsupported distribution: %s", distribution)
	}

	installer := fmt.Sprintf("%s_installer.sh", distribution)
	destinationPath = filepath.Join(destinationPath, installer)

	if err := http.DownloadFile(context.Background(), scriptURL, destinationPath, nil); err != nil {
		return "", fmt.Errorf("downloading script: %w", err)
	}

	if err := os.Chmod(destinationPath, fileio.ExecutablePerms); err != nil {
		return "", fmt.Errorf("modifying script permissions: %w", err)
	}

	return installer, nil
}
