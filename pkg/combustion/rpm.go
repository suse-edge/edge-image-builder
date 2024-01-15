package combustion

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	userRPMsDir         = "rpms"
	modifyRPMScriptName = "10-rpm-install.sh"
	rpmComponentName    = "RPM"
	combustionBasePath  = "/dev/shm/combustion/config"
	createRepoExec      = "/usr/bin/createrepo"
	createRepoLog       = "createrepo.log"
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

	repoName, packages, err := handleRPMs(ctx)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("handling rpms: %w", err)
	}

	script, err := writeRPMScript(ctx, repoName, packages)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("writing the RPM install script %s: %w", modifyRPMScriptName, err)
	}

	log.AuditComponentSuccessful(rpmComponentName)
	return []string{script}, nil
}

func handleRPMs(ctx *image.Context) (repoName string, pkgToInstall []string, err error) {
	if isResolutionNeeded(ctx) {
		repoName, pkgToInstall, err = resolveToRPMRepo(ctx)
		if err != nil {
			return "", nil, fmt.Errorf("resolving rpms to a rpm repository: %w", err)
		}
	} else {
		pkgToInstall, err = rpm.CopyRPMs(generateComponentPath(ctx, userRPMsDir), ctx.CombustionDir)
		if err != nil {
			log.AuditComponentFailed(rpmComponentName)
			return "", nil, fmt.Errorf("moving individual rpm files: %w", err)
		}
	}

	return repoName, pkgToInstall, nil
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

// determine whether package/rpm dependency resolution is needed
func isResolutionNeeded(ctx *image.Context) bool {
	pkg := ctx.ImageDefinition.OperatingSystem.Packages

	if len(pkg.AdditionalRepos) > 0 {
		// Packages/RPMs requested from third party repositories
		return true
	}

	if pkg.RegCode != "" {
		// Packages/RPMs requested from PackageHub
		return true
	}
	return false
}

func resolveToRPMRepo(ctx *image.Context) (repoName string, packages []string, err error) {
	var rpmDir string
	if isComponentConfigured(ctx, userRPMsDir) {
		rpmDir = generateComponentPath(ctx, userRPMsDir)
	}

	repoPath, packages, err := ctx.RPMResolver.Resolve(&ctx.ImageDefinition.OperatingSystem.Packages, rpmDir, ctx.CombustionDir)
	if err != nil {
		return "", nil, fmt.Errorf("resolving rpm/package dependencies: %w", err)
	}

	if err = createRPMRepo(repoPath, ctx.BuildDir); err != nil {
		return "", nil, fmt.Errorf("creating resolved rpm repository: %w", err)
	}

	return filepath.Base(repoPath), packages, nil
}

func createRPMRepo(path, logOut string) error {
	zap.S().Infof("Creating RPM repository from '%s'", path)

	logFile, err := os.Create(filepath.Join(logOut, createRepoLog))
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
	return err
}

func prepareRepoCommand(path string, w io.Writer) *exec.Cmd {
	cmd := exec.Command(createRepoExec, path)
	cmd.Stdout = w
	cmd.Stderr = w

	return cmd
}

func writeRPMScript(ctx *image.Context, repoName string, packages []string) (string, error) {
	if len(packages) == 0 {
		return "", fmt.Errorf("package list cannot be empty")
	}

	values := struct {
		RepoPath string
		RepoName string
		PKGList  string
	}{
		RepoPath: filepath.Join(combustionBasePath, repoName),
		RepoName: repoName,
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
