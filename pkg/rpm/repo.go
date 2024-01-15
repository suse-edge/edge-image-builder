package rpm

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

const (
	createRepoExec = "/usr/bin/createrepo"
	createRepoLog  = "createrepo.log"
)

type RepoCreator struct {
	logOut string
}

func NewRepoCreator(logOut string) *RepoCreator {
	return &RepoCreator{
		logOut: logOut,
	}
}

func (r *RepoCreator) Create(path string) error {
	zap.S().Infof("Creating RPM repository from '%s'", path)

	logFile, err := os.Create(filepath.Join(r.logOut, createRepoLog))
	if err != nil {
		return fmt.Errorf("generating createrepo log file: %w", err)
	}
	defer logFile.Close()

	cmd := prepareRepoCommand(path, logFile)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running createrepo: %w", err)
	}

	zap.L().Info("RPM repository created successfully")
	return nil
}

func prepareRepoCommand(path string, w io.Writer) *exec.Cmd {
	cmd := exec.Command(createRepoExec, path)
	cmd.Stdout = w
	cmd.Stderr = w

	return cmd
}
