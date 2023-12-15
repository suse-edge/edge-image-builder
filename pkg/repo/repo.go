package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/repo/resolver"
	"go.uber.org/zap"
)

const (
	createRepoExec = "/usr/bin/createrepo"
	createRepoLog  = "createrepo-%s.log"
)

// Create creates an RPM repository consisting of the packages/rpms provided
// by the user in the image context and their dependencies. Returns the path
// to the repository and a list of packages that are ready to be installed.
func Create(ctx *image.Context, reslv resolver.Resolver, out string) (string, []string, error) {
	path, pkgs, err := reslv.Resolve(out)
	if err != nil {
		return "", nil, fmt.Errorf("resolving package dependencies: %w", err)
	}

	zap.L().Sugar().Infof("Creating RPM repository from '%s'", path)
	if err := createRPMRepo(path, ctx.BuildDir); err != nil {
		return "", nil, fmt.Errorf("creating rpm repository: %w", err)
	}

	zap.L().Info("RPM repository created successfully")
	return path, pkgs, nil
}

func createRPMRepo(path, logOut string) error {
	cmd, logfile, err := prepareRepoCommand(path, logOut)
	if err != nil {
		return fmt.Errorf("preparing createrepo command: %w", err)
	}
	defer logfile.Close()

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running createrepo: %w", err)
	}

	return err
}

func prepareRepoCommand(path, logOut string) (*exec.Cmd, *os.File, error) {
	logFile, err := generateCreateRepoLog(logOut)
	if err != nil {
		return nil, nil, fmt.Errorf("generating createrepo log file: %w", err)
	}

	cmd := exec.Command(createRepoExec, path)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd, logFile, nil
}

func generateCreateRepoLog(out string) (*os.File, error) {
	timestamp := time.Now().Format("Jan02_15-04-05")
	filename := fmt.Sprintf(createRepoLog, timestamp)
	logFilename := filepath.Join(out, filename)

	logFile, err := os.Create(logFilename)
	if err != nil {
		return nil, fmt.Errorf("creating log file: %w", err)
	}
	zap.L().Sugar().Debugf("log file created: %s", logFilename)

	return logFile, err
}
