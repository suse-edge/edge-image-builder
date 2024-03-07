package combustion

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
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
	userGPGsDir         = "gpg-keys"
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

	packages := &ctx.ImageDefinition.OperatingSystem.Packages
	if packages.NoGPGCheck {
		log.Audit("WARNING: Running EIB with disabled GPG validation is intended for development purposes only")
		zap.S().Warn("Disabling GPG validation for the EIB RPM resolver")
	}

	// package list specified without either a sccRegistrationCode or an additionalRepos entry
	if len(packages.PKGList) > 0 && (packages.RegCode == "" && len(packages.AdditionalRepos) == 0) {
		log.Audit("WARNING: No SUSE registration code or additional repositories provided, package resolution may fail if you're using SLE Micro as the base image")
		zap.S().Warn("Detected packages for installation with no sccRegistrationCode or additionalRepos provided")
	}

	var localRPMConfig *image.LocalRPMConfig
	if isComponentConfigured(ctx, userRPMsDir) {
		rpmDir := RPMsPath(ctx)
		localRPMConfig = &image.LocalRPMConfig{
			RPMPath: rpmDir,
		}

		gpgPath := GPGKeysPath(ctx)
		_, err := os.Stat(gpgPath)
		switch {
		case err == nil:
			if !packages.NoGPGCheck {
				localRPMConfig.GPGKeysPath = gpgPath
			} else {
				log.AuditComponentFailed(rpmComponentName)
				return nil, fmt.Errorf("found existing '%s' directory, but GPG validation is disabled", userGPGsDir)
			}
		case errors.Is(err, fs.ErrNotExist):
			if !packages.NoGPGCheck {
				log.AuditComponentFailed(rpmComponentName)
				return nil, fmt.Errorf("GPG validation is enabled, but '%s' directory is missing for side-loaded RPMs", userGPGsDir)
			}
		case err != nil:
			log.AuditComponentFailed(rpmComponentName)
			return nil, fmt.Errorf("describing GPG directory at '%s': %w", gpgPath, err)
		}
	}

	log.Audit("Resolving package dependencies...")
	repoPath, pkgsList, err := ctx.RPMResolver.Resolve(packages, localRPMConfig, ctx.CombustionDir)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("resolving rpm/package dependencies: %w", err)
	}

	if err = ctx.RPMRepoCreator.Create(repoPath); err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("creating resolved rpm repository: %w", err)
	}

	script, err := writeRPMScript(ctx, repoPath, pkgsList)
	if err != nil {
		log.AuditComponentFailed(rpmComponentName)
		return nil, fmt.Errorf("writing the RPM install script %s: %w", modifyRPMScriptName, err)
	}

	log.AuditComponentSuccessful(rpmComponentName)
	return []string{script}, nil
}

// SkipRPMComponent determines whether RPM configuration is needed
func SkipRPMComponent(ctx *image.Context) bool {
	pkg := ctx.ImageDefinition.OperatingSystem.Packages

	if isComponentConfigured(ctx, userRPMsDir) {
		// isComponentConfigured will indicate if the directory exists, but not
		// if there are RPMs in there. If there aren't any, it is still possible to
		// continue if there have been packages specified in the definition.
		rpmsDir := RPMsPath(ctx)

		dirListing, err := os.ReadDir(rpmsDir)
		if err != nil {
			zap.S().Errorf("checking for side-loaded RPMs: %s", err)
			return true
		}

		// Simply look for at least one .rpm file, the actual amount doesn't matter
		foundRpm := false
		for _, foundFile := range dirListing {
			if filepath.Ext(foundFile.Name()) == ".rpm" {
				foundRpm = true
				break
			}
		}

		if !foundRpm && len(pkg.PKGList) == 0 {
			// Rare case where the rpms directory is specified but empty and no packages
			// are listed. Without this, RPM resolution will trigger and error out about there
			// being "Too few arguments".
			// Ideally, this should probably be done in the validation step, but
			// there is already a precedent for considering this in the custom files handling.
			// For simplicity in solving #242 for the 1.0 release, issue #276 has been created
			// to ensure this logic gets revisited when we get some time to readdress things on
			// a larger scale. jdob, Mar 7, 2024
			return true
		}

		// User provided standalone or third party RPMs, so do not skip the RPM component
		return false
	}
	if len(pkg.PKGList) > 0 {
		// User provided PackageHub or third party packages, so do not skip the RPM component
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

func RPMsPath(ctx *image.Context) string {
	return generateComponentPath(ctx, userRPMsDir)
}

func GPGKeysPath(ctx *image.Context) string {
	rpmDir := RPMsPath(ctx)
	return filepath.Join(rpmDir, userGPGsDir)
}
