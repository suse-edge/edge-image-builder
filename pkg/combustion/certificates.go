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
	certsComponentName = "certificates"
	certsScriptName    = "07-certificates.sh"
	certsConfigDir     = "certificates"
)

//go:embed templates/07-certificates.sh.tpl
var certsScriptTemplate string

func configureCertificates(ctx *image.Context) ([]string, error) {
	if !isComponentConfigured(ctx, certsConfigDir) {
		log.AuditComponentSkipped(certsComponentName)
		zap.S().Info("skipping certificate configuration, no certificates provided")
		return nil, nil
	}

	if err := copyCertificates(ctx); err != nil {
		log.AuditComponentFailed(certsComponentName)
		return nil, err
	}

	if err := writeCertificatesScript(ctx); err != nil {
		log.AuditComponentFailed(certsComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(certsComponentName)
	return []string{certsScriptName}, nil
}

func copyCertificates(ctx *image.Context) error {
	srcDir := filepath.Join(ctx.ImageConfigDir, certsConfigDir)
	destDir := filepath.Join(ctx.CombustionDir, certsConfigDir)

	dirEntries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("reading the certificates directory at %s: %w", srcDir, err)
	}

	if len(dirEntries) == 0 {
		return fmt.Errorf("no certificates found in directory %s", srcDir)
	}

	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("creating certificates directory '%s': %w", destDir, err)
	}

	if err := fileio.CopyFiles(srcDir, destDir, ".pem", false); err != nil {
		return fmt.Errorf("copying pem files: %w", err)
	}

	if err := fileio.CopyFiles(srcDir, destDir, ".crt", false); err != nil {
		return fmt.Errorf("copying certificates: %w", err)
	}

	return nil
}

func writeCertificatesScript(ctx *image.Context) error {
	destFilename := filepath.Join(ctx.CombustionDir, certsScriptName)

	values := struct {
		CertificatesDir string
	}{
		CertificatesDir: certsConfigDir,
	}
	data, err := template.Parse(certsScriptName, certsScriptTemplate, &values)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", certsScriptName, err)
	}

	if err := os.WriteFile(destFilename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", destFilename, err)
	}

	return nil
}
