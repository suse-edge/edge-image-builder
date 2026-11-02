package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	osFilesComponentName = "os files"
	osFilesConfigDir     = "os-files"
	osFilesScriptName    = "19-copy-os-files.sh"
)

var (
	//go:embed templates/19-copy-os-files.sh.tpl
	osFilesScript string
)

// osFilesStagingDir determines where the user provided files are staged.
//
// For image builds this is the artefacts directory. Combustion copies the entire combustion
// directory into a RAM disk before running, so staging there would limit the total size of
// the os-files content to half of the booted system's memory.
//
// Config drives remain staged in the combustion directory, as the artefacts directory of a
// generated drive is currently unreachable at combustion time. The combustion script mounts
// the artefacts by the "INSTALL" filesystem label, whereas generated drives are labelled
// "COMBUSTION".
func osFilesStagingDir(ctx *image.Context) (stagingDir, runtimePath string) {
	if ctx.IsConfigDrive {
		return ctx.CombustionDir, "./" + osFilesConfigDir
	}

	return ctx.ArtefactsDir, prependArtefactPath(osFilesConfigDir)
}

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
	stagingDir, _ := osFilesStagingDir(ctx)

	srcDirectory := filepath.Join(ctx.ImageConfigDir, osFilesConfigDir)
	destDirectory := filepath.Join(stagingDir, osFilesConfigDir)

	dirEntries, err := os.ReadDir(srcDirectory)
	if err != nil {
		return fmt.Errorf("reading the os files directory at %s: %w", srcDirectory, err)
	}

	// If the directory exists but there's nothing in it, consider it an error case
	if len(dirEntries) == 0 {
		return fmt.Errorf("no files found in directory %s", srcDirectory)
	}

	if err := fileio.CopyFiles(srcDirectory, destDirectory, "", true, nil); err != nil {
		return fmt.Errorf("copying os-files: %w", err)
	}

	return nil
}

func writeOSFilesScript(ctx *image.Context) error {
	_, runtimePath := osFilesStagingDir(ctx)

	values := struct {
		FilesPath string
	}{
		FilesPath: runtimePath,
	}

	data, err := template.Parse(osFilesScriptName, osFilesScript, &values)
	if err != nil {
		return fmt.Errorf("parsing os files script template: %w", err)
	}

	osFilesScriptFilename := filepath.Join(ctx.CombustionDir, osFilesScriptName)

	return os.WriteFile(osFilesScriptFilename, []byte(data), fileio.ExecutablePerms)
}
