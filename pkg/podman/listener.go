package podman

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/log"
	"go.uber.org/zap"
)

const (
	podmanArgsBase        = "--log-level=debug system service -t 0"
	podmanExec            = "/usr/bin/podman"
	podmanListenerLogFile = "podman-system-service.log"
	podmanSocketPath      = "/run/podman/podman.sock"
)

// creates a listening service that answers API calls for Podman (https://docs.podman.io/en/v4.8.3/markdown/podman-system-service.1.html)
// only way to start the service from within a container - https://github.com/containers/podman/tree/v4.8.3/pkg/bindings#starting-the-service-manually
func setupAPIListener(out string) error {
	log.AuditInfo("Setting up Podman API listener...")

	logFile, err := os.Create(filepath.Join(out, podmanListenerLogFile))
	if err != nil {
		return fmt.Errorf("creating podman listener log file: %w", err)
	}

	defer logFile.Close()

	cmd := preparePodmanCommand(logFile)
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error running podman system service: %w", err)
	}

	return waitForPodmanSock()
}

func preparePodmanCommand(out io.Writer) *exec.Cmd {
	args := strings.Split(podmanArgsBase, " ")
	cmd := exec.Command(podmanExec, args...)
	cmd.Stdout = out
	cmd.Stderr = out

	return cmd
}

func waitForPodmanSock() error {
	const (
		retries      = 5
		sleepSeconds = 3
	)

	zap.S().Infof("Waiting for '%s' to be created", podmanSocketPath)
	for i := 0; i < retries; i++ {
		if _, err := os.Stat(podmanSocketPath); err == nil {
			zap.S().Infof("'%s' file has been created successfully", podmanSocketPath)
			return nil
		}

		zap.S().Infof("'%s' file is not yet created, retrying in %d seconds", podmanSocketPath, sleepSeconds)
		time.Sleep(sleepSeconds * time.Second)
	}

	return fmt.Errorf("'%s' file was not created in the expected time", podmanSocketPath)
}
