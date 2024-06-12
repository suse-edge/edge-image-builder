package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"go.uber.org/zap"
)

const (
	osFilesComponentName = "os files"
	osFilesConfigDir     = "os-files"
	osFilesScriptName    = "19-copy-os-files.sh"
	osFilesLogFile       = "copy-os-files.log"
)

var (
	//go:embed templates/19-copy-os-files.sh
	osFilesScript string
)

func configureOSFiles(ctx *image.Context) ([]string, error) {
	if !isComponentConfigured(ctx, osFilesConfigDir) {
		log.AuditComponentSkipped(osFilesComponentName)
		zap.S().Info("skipping os files component, no files provided")
		return nil, nil
	}

	if err := copyOSFiles(ctx); err != nil {
		log.AuditComponentFailed(osFilesComponentName)
		return nil, err
	}

	if err := writeOSFilesScript(ctx); err != nil {
		log.AuditComponentFailed(osFilesComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(osFilesComponentName)
	return []string{osFilesScriptName}, nil
}

func copyOSFiles(ctx *image.Context) error {
	srcDirectory := filepath.Join(ctx.ImageConfigDir, osFilesConfigDir)
	destDirectory := filepath.Join(ctx.CombustionDir, osFilesConfigDir)

	dirEntries, err := os.ReadDir(srcDirectory)
	if err != nil {
		return fmt.Errorf("reading the os files directory at %s: %w", srcDirectory, err)
	}

	// If the directory exists but there's nothing in it, consider it an error case
	if len(dirEntries) == 0 {
		return fmt.Errorf("no files found in directory %s", srcDirectory)
	}

	logFilename := filepath.Join(ctx.BuildDir, osFilesLogFile)
	logFile, err := os.Create(logFilename)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}

	defer func() {
		if err = logFile.Close(); err != nil {
			zap.S().Warnf("failed to close copy os-files log file properly: %s", err)
		}
	}()

	if err := fileio.CopyFiles(srcDirectory, destDirectory, "", true); err != nil {
		return fmt.Errorf("running copy os-files command: %w", err)
	}

	return nil
}

func writeOSFilesScript(ctx *image.Context) error {
	osFilesScriptFilename := filepath.Join(ctx.CombustionDir, osFilesScriptName)

	if err := os.WriteFile(osFilesScriptFilename, []byte(osFilesScript), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing os files script %s: %w", osFilesScriptFilename, err)
	}

	return nil
}
