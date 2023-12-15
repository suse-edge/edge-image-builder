package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
)

const (
	userRPMsDir         = "rpms"
	modifyRPMScriptName = "10-rpm-install.sh"
	rpmComponentName    = "RPM"
)

//go:embed templates/10-rpm-install.sh.tpl
var modifyRPMScript string

func configureRPMs(ctx *image.Context) ([]string, error) {
	if !isComponentConfigured(ctx, userRPMsDir) {
		log.AuditComponentSkipped(rpmComponentName)
		return nil, nil
	}

	rpmSourceDir := generateComponentPath(ctx, userRPMsDir)

	rpmFileNames, err := getRPMFileNames(rpmSourceDir)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("getting RPM file names: %w", err)
	}

	err = copyRPMs(rpmSourceDir, ctx.CombustionDir, rpmFileNames)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("copying RPMs over: %w", err)
	}

	script, err := writeRPMScript(ctx, rpmFileNames)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("writing the RPM install script %s: %w", modifyRPMScriptName, err)
	}

	log.AuditComponentSuccessful(rpmComponentName)
	return []string{script}, nil
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
