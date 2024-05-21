package combustion

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
	networkConfigScriptName = "05-configure-network.sh"
	networkCustomScriptName = "configure-network.sh"
)

//go:embed templates/05-configure-network.sh.tpl
var configureNetworkScript string

// Configures the network component if enabled.
//
//  1. Copies the nmc executable
//  2. Copies a custom network configuration script if provided
//  3. Generates network configurations and writes the configuration script template otherwise
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
//	└── 05-configure-network.sh
func (c *Combustion) configureNetwork(ctx *image.Context) (scripts []string, err error) {
	zap.S().Info("Configuring network component...")

	if !isComponentConfigured(ctx, networkConfigDir) {
		log.AuditComponentSkipped(networkComponentName)
		zap.S().Info("Skipping network component, configuration is not provided")
		return nil, nil
	}

	defer func() {
		logComponentStatus(networkComponentName, err)
	}()

	networkPath := generateComponentPath(ctx, networkConfigDir)

	entries, err := os.ReadDir(networkPath)
	if err != nil {
		return nil, fmt.Errorf("reading network directory: %w", err)
	} else if len(entries) == 0 {
		return nil, fmt.Errorf("network directory is present but empty")
	}

	if err = c.installNetworkConfigurator(ctx); err != nil {
		return nil, fmt.Errorf("installing configurator: %w", err)
	}

	customScript := filepath.Join(networkPath, networkCustomScriptName)
	combustionScript := filepath.Join(ctx.CombustionDir, networkConfigScriptName)
	scripts = append(scripts, networkConfigScriptName)

	// Copy custom network script if provided.
	// Proceed with generating configuration otherwise.
	err = fileio.CopyFile(customScript, combustionScript, fileio.ExecutablePerms)
	if err == nil {
		return scripts, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("copying custom network script: %w", err)
	}

	if err = c.generateNetworkConfig(ctx); err != nil {
		return nil, fmt.Errorf("generating network config: %w", err)
	}

	if err = writeNetworkConfigurationScript(combustionScript); err != nil {
		return nil, fmt.Errorf("writing network configuration script: %w", err)
	}

	return scripts, nil
}

func (c *Combustion) generateNetworkConfig(ctx *image.Context) error {
	const networkConfigLogFile = "network-config.log"

	logFilename := filepath.Join(ctx.BuildDir, networkConfigLogFile)
	logFile, err := os.Create(logFilename)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}

	defer func() {
		if err = logFile.Close(); err != nil {
			zap.S().Warnf("Failed to close network log file properly: %s", err)
		}
	}()

	configDir := generateComponentPath(ctx, networkConfigDir)
	outputDir := filepath.Join(ctx.CombustionDir, networkConfigDir)

	return c.NetworkConfigGenerator.GenerateNetworkConfig(configDir, outputDir, logFile)
}

func (c *Combustion) installNetworkConfigurator(ctx *image.Context) error {
	sourcePath := "/usr/bin/nmc"
	installPath := filepath.Join(ctx.CombustionDir, nmcExecutable)

	return c.NetworkConfiguratorInstaller.InstallConfigurator(sourcePath, installPath)
}

func writeNetworkConfigurationScript(scriptPath string) error {
	values := struct {
		ConfigDir string
	}{
		ConfigDir: networkConfigDir,
	}

	data, err := template.Parse(networkConfigScriptName, configureNetworkScript, &values)
	if err != nil {
		return fmt.Errorf("parsing network template: %w", err)
	}

	if err = os.WriteFile(scriptPath, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing network script: %w", err)
	}

	return nil
}
