package resolver

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	prepareTarballScriptName = "prepare-resolver-base-tarball-image.sh"
	prepareTarballScriptLog  = "prepare-resolver-base-tarball-image.log"
	tarballName              = "resolver-base-tarball-image.tar.gz"
	tarballImgRef            = "resolver-base-tarball-image"
)

//go:embed templates/prepare-tarball.sh.tpl
var prepareTraballTemplate string

type ImageImporter interface {
	Import(tarball, ref string) error
}

type TarballImageBuilder struct {
	// dir from where the image builder will work
	dir string
	// path to the ISO/RAW file from which a tarball will be created
	imgPath string
	// type of the image that will be used as base (either ISO or RAW)
	imgType string
	// imgImporter used to import the tarball archive as a container image
	imgImporter ImageImporter
}

func NewTarballBuilder(workDir, imgPath, imgType string, importer ImageImporter) *TarballImageBuilder {
	return &TarballImageBuilder{
		dir:         workDir,
		imgPath:     imgPath,
		imgType:     imgType,
		imgImporter: importer,
	}
}

func (t *TarballImageBuilder) Build() (string, error) {
	zap.L().Info("Building tarball image...")
	defer os.RemoveAll(t.getTarballImgDir())

	if err := t.prepareTarball(); err != nil {
		return "", fmt.Errorf("preparing the tarball image env: %w", err)
	}

	if err := t.writeTarballImageScript(); err != nil {
		return "", fmt.Errorf("writing the tarball image script: %w", err)
	}

	if err := t.runTarballImageScript(); err != nil {
		return "", fmt.Errorf("running the tarball image script: %w", err)
	}

	tarballPath := filepath.Join(t.getTarballImgDir(), tarballName)
	if err := t.imgImporter.Import(tarballPath, tarballImgRef); err != nil {
		return "", fmt.Errorf("importing the tarball image: %w", err)
	}

	zap.L().Info("Tarball image build successful")
	return tarballImgRef, nil
}

func (t *TarballImageBuilder) prepareTarball() error {
	tarballImgDir := t.getTarballImgDir()
	if err := os.MkdirAll(tarballImgDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating %s dir: %w", tarballImgDir, err)
	}

	// copy user provided image so that the builder can
	// safely work on the copy without worrying that it might
	// break the provided image
	if err := fileio.CopyFile(t.imgPath, t.getBaseISOCopyPath(), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("creating work copy of image %s in repo work dir %s: %w", t.imgPath, tarballImgDir, err)
	}

	return nil
}

func (t *TarballImageBuilder) writeTarballImageScript() error {
	values := struct {
		WorkDir     string
		ImgPath     string
		ArchiveName string
		ImgType     string
	}{
		WorkDir:     t.getTarballImgDir(),
		ImgPath:     t.getBaseISOCopyPath(),
		ArchiveName: tarballName,
		ImgType:     t.imgType,
	}

	data, err := template.Parse(prepareTarballScriptName, prepareTraballTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", prepareTarballScriptName, err)
	}

	filename := filepath.Join(t.dir, prepareTarballScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing script %s: %w", filename, err)
	}

	return nil
}

func (t *TarballImageBuilder) runTarballImageScript() error {
	logFile, err := os.Create(filepath.Join(t.dir, prepareTarballScriptLog))
	if err != nil {
		return fmt.Errorf("generating prepare tarball image log file: %w", err)
	}
	defer logFile.Close()

	cmd := t.prepareTarballImageCmd(logFile)
	if err = cmd.Run(); err != nil {
		return fmt.Errorf("run script failure: %w", err)
	}

	return nil
}

func (t *TarballImageBuilder) prepareTarballImageCmd(log io.Writer) *exec.Cmd {
	scriptPath := filepath.Join(t.dir, prepareTarballScriptName)
	cmd := exec.Command(scriptPath)
	cmd.Stdout = log
	cmd.Stderr = log
	return cmd
}

func (t *TarballImageBuilder) getTarballImgDir() string {
	return filepath.Join(t.dir, "resolver-tarball-image")
}

func (t *TarballImageBuilder) getBaseISOCopyPath() string {
	return filepath.Join(t.getTarballImgDir(), filepath.Base(t.imgPath))
}
