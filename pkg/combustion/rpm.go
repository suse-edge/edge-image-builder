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
	"github.com/suse-edge/edge-image-builder/pkg/repo"
	"github.com/suse-edge/edge-image-builder/pkg/repo/resolver"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	userRPMsDir         = "rpms"
	modifyRPMScriptName = "10-rpm-install.sh"
	rpmComponentName    = "RPM"
	combustionBasePath  = "/dev/shm/combustion/config"
)

//go:embed templates/10-rpm-install.sh.tpl
var modifyRPMScript string

func configureRPMs(ctx *image.Context) ([]string, error) {
	if skipRPMconfigre(ctx) {
		log.AuditComponentSkipped(rpmComponentName)
		zap.L().Info("Skipping RPM component. Configuration is not provided")
		return nil, nil
	}

	zap.L().Info("Configuring RPM component...")
	var repoName string
	var packages []string

	// check if there is a need for pkg/rpm dependency resolution
	// if there is no need, then treat RPMs as standalone rpms and do
	// not create an RPM dependency
	if isResolutionNeeded(ctx) {
		reslv, err := resolver.New(ctx)
		if err != nil {
			log.AuditComponentFailed(rpmComponentName)
			return nil, fmt.Errorf("initializing resolver: %w", err)
		}

		repoPath, pkgList, err := repo.Create(ctx, reslv, ctx.CombustionDir)
		if err != nil {
			log.AuditComponentFailed(rpmComponentName)
			return nil, fmt.Errorf("creating rpm repository: %w", err)
		}

		repoName = filepath.Base(repoPath)
		packages = pkgList
	} else {
		rpms, err := rpm.CopyRPMs(generateComponentPath(ctx, userRPMsDir), ctx.CombustionDir)
		if err != nil {
			log.AuditComponentFailed(rpmComponentName)
			return nil, fmt.Errorf("moving single rpm files: %w", err)
		}

		packages = rpms
	}

	script, err := writeRPMScript(ctx, repoName, packages)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("writing the RPM install script %s: %w", modifyRPMScriptName, err)
	}

	log.AuditComponentSuccessful(rpmComponentName)
	return []string{script}, nil
}

func writeRPMScript(ctx *image.Context, repoName string, pkgList []string) (string, error) {
	if len(pkgList) == 0 {
		return "", fmt.Errorf("package list cannot be empty")
	}

	values := struct {
		RepoPath string
		RepoName string
		PKGList  string
	}{
		RepoPath: filepath.Join(combustionBasePath, repoName),
		RepoName: repoName,
		PKGList:  strings.Join(pkgList, " "),
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

// determine whether RPM configuration is needed
func skipRPMconfigre(ctx *image.Context) bool {
	pkg := ctx.ImageDefinition.OperatingSystem.Packages

	if isComponentConfigured(ctx, userRPMsDir) && len(pkg.AddRepos) > 0 {
		// is RPM configured in 'rpms' directory from a third party repository
		return false
	} else if isComponentConfigured(ctx, userRPMsDir) {
		// is RPM configured as a standalone RPM inside of the 'rpms' directory
		return false
	}

	// is package configured for installation from PackageHub
	if len(pkg.PKGList) > 0 && pkg.RegCode != "" {
		return false
	}

	// is package configured for installation from a third party repository
	if len(pkg.AddRepos) > 0 && len(pkg.PKGList) > 0 {
		return false
	}

	return true
}

// determine whether package/rpm dependency resolution is needed
func isResolutionNeeded(ctx *image.Context) bool {
	pkg := ctx.ImageDefinition.OperatingSystem.Packages

	// check if:
	// 1. packages from PackageHub are provided
	// 2. third party packges are provided
	// 3. third party repos for rpms are provided
	if len(pkg.PKGList) > 0 && pkg.RegCode != "" {
		return true
	} else if len(pkg.AddRepos) > 0 && len(pkg.PKGList) > 0 {
		return true
	} else if len(pkg.AddRepos) > 0 {
		return true
	}
	return false
}
