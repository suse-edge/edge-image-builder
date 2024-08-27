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
	FipsPackages   = []string{"patterns-base-fips"}
	FipsKernelArgs = []string{"fips=1"}
)

func configureFips(ctx *image.Context) ([]string, error) {
	fips := ctx.ImageDefinition.OperatingSystem.EnableFips
	if !fips {
		log.AuditComponentSkipped(fipsComponentName)
		return nil, nil
	}

	if err := writeFipsCombustionScript(ctx); err != nil {
		log.AuditComponentFailed(fipsComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(fipsComponentName)
	return []string{fipsScriptName}, nil
}

func writeFipsCombustionScript(ctx *image.Context) error {
	fipsScriptFilename := filepath.Join(ctx.CombustionDir, fipsScriptName)

	if err := os.WriteFile(fipsScriptFilename, []byte(fipsScript), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", fipsScriptFilename, err)
	}
	return nil
}
