package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

const (
	fipsComponentName = "fips"
	fipsScriptName    = "15-fips-setup.sh"
)

var (
	//go:embed templates/15-fips-setup.sh
	fipsScript     string
	FIPSPackages   = []string{"pattern:fips"}
	FIPSKernelArgs = []string{"fips=1"}
)

func configureFIPS(ctx *image.Context) ([]string, error) {
	fips := ctx.ImageDefinition.OperatingSystem.EnableFIPS
	if !fips {
		log.AuditComponentSkipped(fipsComponentName)
		return nil, nil
	}

	if err := writeFIPSCombustionScript(ctx); err != nil {
		log.AuditComponentFailed(fipsComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(fipsComponentName)
	return []string{fipsScriptName}, nil
}

func writeFIPSCombustionScript(ctx *image.Context) error {
	fipsScriptFilename := filepath.Join(ctx.CombustionDir, fipsScriptName)

	if err := os.WriteFile(fipsScriptFilename, []byte(fipsScript), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", fipsScriptFilename, err)
	}
	return nil
}
