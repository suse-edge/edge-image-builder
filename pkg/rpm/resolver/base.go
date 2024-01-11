package resolver

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	prepareBaseScriptName = "prepare-base.sh"
	prepareBaseScriptLog  = "prepare-base.log"
	baseImageArchiveName  = "sle-micro-base.tar.gz"
	baseImageRef          = "slemicro"
)

//go:embed templates/prepare-base.sh.tpl
var prepareBaseTemplate string

func (r *Resolver) buildBase(podman image.Podman) error {
	zap.L().Info("Building base resolver image...")

	defer os.RemoveAll(r.getBaseImgDir())
	if err := r.prepareBase(); err != nil {
		return fmt.Errorf("preparing base image env: %w", err)
	}

	if err := r.writeBaseImageScript(); err != nil {
		return fmt.Errorf("writing base resolver image script: %w", err)
	}

	if err := r.runBaseImageScript(); err != nil {
		return fmt.Errorf("running base resolver image script: %w", err)
	}

	tarballPath := filepath.Join(r.getBaseImgDir(), baseImageArchiveName)
	if err := podman.Import(tarballPath, baseImageRef); err != nil {
		return fmt.Errorf("importing the base image: %w", err)
	}

	zap.L().Info("Base resolver image build successful")
	return nil
}

func (r *Resolver) prepareBase() error {
	baseImgDir := r.getBaseImgDir()
	if err := os.MkdirAll(baseImgDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating %s dir: %w", baseImgDir, err)
	}

	// copy user provided image so that the resolver can
	// safely work on the copy without worrying that it might
	// break other EIB functionality
	if err := fileio.CopyFile(r.imgPath, r.getBaseISOCopyPath(), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating work copy of image %s in repo work dir %s: %w", r.imgPath, baseImgDir, err)
	}

	return nil
}

func (r *Resolver) runBaseImageScript() error {
	logFile, err := os.Create(filepath.Join(r.dir, prepareBaseScriptLog))
	if err != nil {
		return fmt.Errorf("generating prepare base image log file: %w", err)
	}
	defer logFile.Close()

	cmd := r.prepareBaseImageCmd(logFile)
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("run script failure: %w", err)
	}

	return nil
}

func (r *Resolver) prepareBaseImageCmd(log io.Writer) *exec.Cmd {
	scriptPath := filepath.Join(r.dir, prepareBaseScriptName)
	cmd := exec.Command(scriptPath)
	cmd.Stdout = log
	cmd.Stderr = log
	return cmd
}

func (r *Resolver) writeBaseImageScript() error {
	values := struct {
		WorkDir     string
		ImgPath     string
		ArchiveName string
		ImgType     string
	}{
		WorkDir:     r.getBaseImgDir(),
		ImgPath:     r.getBaseISOCopyPath(),
		ArchiveName: baseImageArchiveName,
		ImgType:     r.imgType,
	}

	data, err := template.Parse(prepareBaseScriptName, prepareBaseTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", prepareBaseScriptName, err)
	}

	filename := filepath.Join(r.dir, prepareBaseScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing script %s: %w", filename, err)
	}

	return nil
}

func (r *Resolver) getBaseImgDir() string {
	return filepath.Join(r.dir, "resolver-base-image")
}

func (r *Resolver) getBaseISOCopyPath() string {
	return filepath.Join(r.getBaseImgDir(), filepath.Base(r.imgPath))
}
