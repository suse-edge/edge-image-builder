package podman

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	podmanArgsBase        = "--log-level=debug system service -t 0"
	podmanExec            = "/usr/bin/podman"
	podmanListenerLogFile = "podman-system-service.log"
)

// creates a listening service that answers API calls for Podman (https://docs.podman.io/en/v4.8.2/markdown/podman-system-service.1.html)
// only way to start the service from within a container - https://github.com/containers/podman/tree/v4.8.2/pkg/bindings#starting-the-service-manually
func setupAPIListener(out string) error {
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

	return nil
}

func preparePodmanCommand(out io.Writer) *exec.Cmd {
	args := strings.Split(podmanArgsBase, " ")
	cmd := exec.Command(podmanExec, args...)
	cmd.Stdout = out
	cmd.Stderr = out

	return cmd
}
