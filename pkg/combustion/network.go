package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	networkComponentName = "network"
	nmcExecutable        = "nmc"
	// Used for both input component source and
	// output configurations subdirectory under combustion.
	networkConfigDir        = "network"
	networkConfigScriptName = "configure-network.sh"
)

//go:embed templates/configure-network.sh.tpl
var configureNetworkScript string

// Configures the network component if enabled.
//
//  1. Generates network configurations
//  2. Copies the NMC executable
//  3. Writes the configuration script
//
// Example result file layout:
//
//	combustion
//	├── network
//	│   ├── node1.example.com
//	│   │   ├── eth0.nmconnection
//	│   │   └── eth1.nmconnection
//	│   ├── node2.example.com
//	│   │   └── eth0.nmconnection
//	│   ├── node3.example.com
//	│   │   ├── bond0.nmconnection
//	│   │   └── eth1.nmconnection
//	│   └── host_config.yaml
//	├── nmc
//	└── configure-network.sh
func configureNetwork(ctx *image.Context) ([]string, error) {
	zap.L().Info("Configuring network component...")

	if !isComponentConfigured(ctx, networkConfigDir) {
		log.AuditComponentSkipped(networkComponentName)
		zap.L().Info("Skipping network component. Configuration is not provided")
		return nil, nil
	}

	if err := generateNetworkConfig(ctx); err != nil {
		log.AuditComponentFailed(networkComponentName)
		return nil, fmt.Errorf("generating network config: %w", err)
	}

	if err := installNetworkConfigurator(ctx); err != nil {
		log.AuditComponentFailed(networkComponentName)
		return nil, fmt.Errorf("installing configurator: %w", err)
	}

	scriptName, err := writeNetworkConfigurationScript(ctx)
	if err != nil {
		log.AuditComponentFailed(networkComponentName)
		return nil, fmt.Errorf("writing network configuration script: %w", err)
	}

	log.AuditComponentSuccessful(networkComponentName)
	zap.L().Info("Successfully configured network component")

	return []string{scriptName}, nil
}

func generateNetworkConfig(ctx *image.Context) error {
	logFilename := generateNetworkLogFilename(ctx)
	logFile, err := os.Create(logFilename)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}

	defer func() {
		if err = logFile.Close(); err != nil {
			zap.L().Warn("Failed to close network log file properly", zap.Error(err))
		}
	}()

	configDir := generateComponentPath(ctx, networkConfigDir)
	outputDir := filepath.Join(ctx.CombustionDir, networkConfigDir)

	return ctx.NetworkConfigGenerator.GenerateNetworkConfig(configDir, outputDir, logFile)
}

func generateNetworkLogFilename(ctx *image.Context) string {
	const networkConfigLogFile = "network-config-%s.log"

	timestamp := time.Now().Format("Jan02_15-04-05")
	filename := fmt.Sprintf(networkConfigLogFile, timestamp)

	return filepath.Join(ctx.BuildDir, filename)
}

func installNetworkConfigurator(ctx *image.Context) error {
	sourcePath := "/" // root level of the container image
	installPath := filepath.Join(ctx.CombustionDir, nmcExecutable)

	return ctx.NetworkConfiguratorInstaller.InstallConfigurator(ctx.ImageDefinition.Image.BaseImage, sourcePath, installPath)
}

func writeNetworkConfigurationScript(ctx *image.Context) (string, error) {
	values := struct {
		ConfigDir string
	}{
		ConfigDir: networkConfigDir,
	}

	data, err := template.Parse(networkConfigScriptName, configureNetworkScript, &values)
	if err != nil {
		return "", fmt.Errorf("parsing network template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, networkConfigScriptName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return "", fmt.Errorf("writing network script: %w", err)
	}

	return networkConfigScriptName, nil
}
