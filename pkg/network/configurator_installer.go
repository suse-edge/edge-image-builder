package network

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
)

const nmcDownloadURL = "https://github.com/suse-edge/nm-configurator/releases/download/v0.2.0/nmc-linux-%s"

type ConfiguratorInstaller struct{}

func (ConfiguratorInstaller) InstallConfigurator(imageName, installPath string) error {
	var arch string

	switch {
	case strings.Contains(imageName, "x86_64"):
		arch = "x86_64"
	case strings.Contains(imageName, "aarch64"):
		arch = "aarch64"
	default:
		return fmt.Errorf("failed to determine arch of image %s", imageName)
	}

	file, err := os.Create(installPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	if err = file.Chmod(fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("adjusting permissions: %w", err)
	}

	downloadURL := fmt.Sprintf(nmcDownloadURL, arch)

	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading configurator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("storing configurator: %w", err)
	}

	return nil
}
