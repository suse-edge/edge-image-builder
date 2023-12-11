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
)

const (
	sumaComponentName = "suma"
	sumaScriptName    = "12-suma-registration.sh"
)

//go:embed templates/12-suma-register.sh.tpl
var sumaScript string

func configureSuma(ctx *image.Context) ([]string, error) {
	suma := ctx.ImageDefinition.OperatingSystem.Suma
	if suma.Host == "" {
		log.AuditComponentSkipped(sumaComponentName)
		return nil, nil
	}

	if err := writeSumaCombustionScript(ctx); err != nil {
		log.AuditComponentFailed(sumaComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(sumaComponentName)
	return []string{sumaScriptName}, nil
}

func writeSumaCombustionScript(ctx *image.Context) error {
	sumaScriptFilename := filepath.Join(ctx.CombustionDir, sumaScriptName)

	data, err := template.Parse(sumaScriptName, sumaScript, ctx.ImageDefinition.OperatingSystem.Suma)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", sumaScriptName, err)
	}

	if err := os.WriteFile(sumaScriptFilename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", sumaScriptFilename, err)
	}
	return nil
}
