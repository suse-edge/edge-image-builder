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
	networkConfigdriveComponentName = "network-configdrive"
	networkConfigdriveDir           = "network-configdrive"
	networkConfigdriveScriptName    = "03-configure-network-configdrive.sh"
)

//go:embed templates/03-configure-network-configdrive.sh
var configureNetworkConfigdriveScript string

// Configures the network via configdrive if enabled.
//
//  1. Copies the NMC executable
//  2. Writes the configuration script
//
// Example result file layout:
//
// combustion
// ├── nmc
// └── 03-configure-network-configdrive.sh
func configureNetworkConfigdrive(ctx *image.Context) ([]string, error) {
	if ! ctx.ImageDefinition.OperatingSystem.ConfigDrive {
		log.AuditComponentSkipped(networkConfigdriveComponentName)
		return nil, nil
	}

	if err := installNetworkConfigurator(ctx); err != nil {
		log.AuditComponentFailed(networkConfigdriveComponentName)
		return nil, fmt.Errorf("installing configurator: %w", err)
	}

	scriptName, err := writeNetworkConfigdriveScript(ctx)
	if err != nil {
		log.AuditComponentFailed(networkConfigdriveComponentName)
		return nil, fmt.Errorf("writing network configuration script: %w", err)
	}

	log.AuditComponentSuccessful(networkConfigdriveComponentName)
	zap.S().Info("Successfully configured network component")

	return []string{scriptName}, nil
}

func writeNetworkConfigdriveScript(ctx *image.Context) (string, error) {
	filename := filepath.Join(ctx.CombustionDir, networkConfigdriveScriptName)
	if err := os.WriteFile(filename, []byte(configureNetworkConfigdriveScript), fileio.ExecutablePerms); err != nil {
		return "", fmt.Errorf("writing network script: %w", err)
	}
	return networkConfigdriveScriptName, nil
}
