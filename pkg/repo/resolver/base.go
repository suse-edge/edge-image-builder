package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	prepareBaseScriptName = "prepare-base.sh"
	baseImageArchiveName  = "sle-micro-base.tar.gz"
	baseImageRef          = "slemicro"
)

//go:embed scripts/prepare-base.sh.tpl
var prepareBaseTemplate string

func (r *resolver) buildBase() error {
	zap.L().Info("Building base resolver image...")

	defer os.RemoveAll(r.getBaseImgDir())
	if err := r.prepareBase(); err != nil {
		return fmt.Errorf("preparing base image env: %w", err)
	}

	if err := r.writeBaseImageScript(); err != nil {
		return fmt.Errorf("writing base resolver image script: %w", err)
	}

	cmd := r.prepareBaseImageCmd()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("running the prepare base image script: %w", err)
	}

	tarballPath := filepath.Join(r.getBaseImgDir(), baseImageArchiveName)
	if err := r.podman.Import(tarballPath, baseImageRef); err != nil {
		return fmt.Errorf("importing the base image: %w", err)
	}

	zap.L().Info("Base resolver image build successful")
	return nil
}

func (r *resolver) prepareBase() error {
	baseImgDir := r.getBaseImgDir()
	if err := os.MkdirAll(baseImgDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating %s dir: %w", baseImgDir, err)
	}

	// copy user provided image so that the resolver can
	// safely work on the copy without having worrying that it might
	// break other EIB functionality
	if err := fileio.CopyFile(r.imgPath, r.getBaseISOCopyPath(), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating work copy of image %s in repo work dir %s: %w", r.imgPath, baseImgDir, err)
	}

	return nil
}

func (r *resolver) writeBaseImageScript() error {
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

func (r *resolver) prepareBaseImageCmd() *exec.Cmd {
	scriptPath := filepath.Join(r.dir, prepareBaseScriptName)
	cmd := exec.Command(scriptPath)
	return cmd
}

func (r *resolver) getBaseImgDir() string {
	return filepath.Join(r.dir, "base-image")
}

func (r *resolver) getBaseISOCopyPath() string {
	return filepath.Join(r.getBaseImgDir(), filepath.Base(r.imgPath))
}
