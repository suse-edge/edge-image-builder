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
	proxyComponentName = "proxy"
	proxyScriptName    = "08-proxy-setup.sh"
)

//go:embed templates/08-proxy-setup.sh.tpl
var proxyScript string

func configureProxy(ctx *image.Context) ([]string, error) {
	proxy := ctx.ImageDefinition.OperatingSystem.Proxy
	if proxy.HttpProxy == "" && proxy.HttpsProxy == "" {
		log.AuditComponentSkipped(proxyComponentName)
		return nil, nil
	}

	if err := writeProxyCombustionScript(ctx); err != nil {
		log.AuditComponentFailed(proxyComponentName)
		return nil, err
	}

	log.AuditComponentSuccessful(proxyComponentName)
	return []string{proxyScriptName}, nil
}

func writeProxyCombustionScript(ctx *image.Context) error {
	proxyScriptFilename := filepath.Join(ctx.CombustionDir, proxyScriptName)

	data, err := template.Parse(proxyScriptName, proxyScript, ctx.ImageDefinition.OperatingSystem.Proxy)
	if err != nil {
		return fmt.Errorf("applying template to %s: %w", proxyScriptName, err)
	}

	if err := os.WriteFile(proxyScriptFilename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing file %s: %w", proxyScriptFilename, err)
	}
	return nil
}
