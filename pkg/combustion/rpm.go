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
	"go.uber.org/zap"
)

const (
	userRPMsDir         = "rpms"
	modifyRPMScriptName = "10-rpm-install.sh"
	rpmComponentName    = "RPM"
)

//go:embed templates/10-rpm-install.sh.tpl
var modifyRPMScript string

func configureRPMs(ctx *image.Context) ([]string, error) {
	if SkipRPMComponent(ctx) {
		log.AuditComponentSkipped(rpmComponentName)
		zap.L().Info("Skipping RPM component. Configuration is not provided")
		return nil, nil
	}

	zap.L().Info("Configuring RPM component...")

	var rpmDir string
	if isComponentConfigured(ctx, userRPMsDir) {
		rpmDir = generateComponentPath(ctx, userRPMsDir)
	}

	repoPath, packages, err := ctx.RPMResolver.Resolve(&ctx.ImageDefinition.OperatingSystem.Packages, rpmDir, ctx.CombustionDir)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("resolving rpm/package dependencies: %w", err)
	}

	if err = ctx.RPMRepoCreator.Create(repoPath); err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("creating resolved rpm repository: %w", err)
	}

	script, err := writeRPMScript(ctx, repoPath, packages)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("writing the RPM install script %s: %w", modifyRPMScriptName, err)
	}

	log.AuditComponentSuccessful(rpmComponentName)
	return []string{script}, nil
}

// determine whether RPM configuration is needed
func SkipRPMComponent(ctx *image.Context) bool {
	pkg := ctx.ImageDefinition.OperatingSystem.Packages

	if isComponentConfigured(ctx, userRPMsDir) {
		// User provided standalone or third party RPMs
		return false
	}
	if len(pkg.PKGList) > 0 {
		// User provided PackageHub or third party packages
		return false
	}

	return true
}

func writeRPMScript(ctx *image.Context, repoPath string, packages []string) (string, error) {
	if len(packages) == 0 {
		return "", fmt.Errorf("package list cannot be empty")
	}

	if repoPath == "" {
		return "", fmt.Errorf("path to RPM repository cannot be empty")
	}

	values := struct {
		RepoName string
		PKGList  string
	}{
		RepoName: filepath.Base(repoPath),
		PKGList:  strings.Join(packages, " "),
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
