package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	modifyRPMScriptName = "10-rpm-install.sh"
)

//go:embed templates/10-rpm-install.sh.tpl
var modifyRPMScript string

func configureRPMs(ctx *image.Context) ([]string, error) {
	rpmSourceDir, err := generateRPMPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating RPM path: %w", err)
	}
	// Only proceed with processing the RPMs if the directory exists
	if rpmSourceDir == "" {
		return nil, nil
	}

	rpmFileNames, err := getRPMFileNames(rpmSourceDir)
	if err != nil {
		return nil, fmt.Errorf("getting RPM file names: %w", err)
	}

	err = copyRPMs(rpmSourceDir, ctx.CombustionDir, rpmFileNames)
	if err != nil {
		return nil, fmt.Errorf("copying RPMs over: %w", err)
	}

	script, err := writeRPMScript(ctx, rpmFileNames)
	if err != nil {
		return nil, fmt.Errorf("writing the RPM install script %s: %w", modifyRPMScriptName, err)
	}

	return []string{script}, nil
}

func getRPMFileNames(rpmSourceDir string) ([]string, error) {
	var rpmFileNames []string

	rpms, err := os.ReadDir(rpmSourceDir)
	if err != nil {
		return nil, fmt.Errorf("reading RPM source dir: %w", err)
	}

	for _, rpmFile := range rpms {
		if filepath.Ext(rpmFile.Name()) == ".rpm" {
			rpmFileNames = append(rpmFileNames, rpmFile.Name())
		}
	}

	if len(rpmFileNames) == 0 {
		return nil, fmt.Errorf("no RPMs found")
	}

	return rpmFileNames, nil
}

func copyRPMs(rpmSourceDir string, rpmDestDir string, rpmFileNames []string) error {
	if rpmDestDir == "" {
		return fmt.Errorf("RPM destination directory cannot be empty")
	}
	for _, rpm := range rpmFileNames {
		sourcePath := filepath.Join(rpmSourceDir, rpm)
		destPath := filepath.Join(rpmDestDir, rpm)

		err := fileio.CopyFile(sourcePath, destPath, fileio.NonExecutablePerms)
		if err != nil {
			return fmt.Errorf("copying file %s: %w", sourcePath, err)
		}
	}

	return nil
}

func writeRPMScript(ctx *image.Context, rpmFileNames []string) (string, error) {
	values := struct {
		RPMs string
	}{
		RPMs: strings.Join(rpmFileNames, " "),
	}

	data, err := template.Parse(modifyRPMScriptName, modifyRPMScript, &values)
	if err != nil {
		return "", fmt.Errorf("parsing RPM script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, modifyRPMScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		return "", fmt.Errorf("writing RPM script: %w", err)
	}

	return modifyRPMScriptName, nil
}

func generateRPMPath(ctx *image.Context) (string, error) {
	rpmSourceDir := filepath.Join(ctx.ImageConfigDir, "rpms")
	_, err := os.Stat(rpmSourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("checking for RPM directory at %s: %w", rpmSourceDir, err)
	}

	return rpmSourceDir, nil
}
