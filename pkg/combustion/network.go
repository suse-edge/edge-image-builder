package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	nmcExecutable           = "nmc"
	nmcConfigDir            = "config"
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
//	│   ├── config
//	│   │   ├── node1.example.com
//	│   │   │   ├── eth0.nmconnection
//	│   │   │   └── eth1.nmconnection
//	│   │   ├── node2.example.com
//	│   │   │   └── eth0.nmconnection
//	│   │   ├── node3.example.com
//	│   │   │   ├── bond0.nmconnection
//	│   │   │   └── eth1.nmconnection
//	│   │   └── host_config.yaml
//	│   └── nmc
//	└── configure-network.sh
func configureNetwork(ctx *image.Context) ([]string, error) {
	if !isComponentConfigured(ctx, networkConfigDir) {
		return nil, nil
	}

	zap.L().Info("Configuring network component...")

	if err := generateNetworkConfig(ctx); err != nil {
		return nil, fmt.Errorf("generating network config: %w", err)
	}

	if err := writeNMCExecutable(ctx); err != nil {
		return nil, fmt.Errorf("writing nmc executable: %w", err)
	}

	scriptName, err := writeNetworkConfigurationScript(ctx)
	if err != nil {
		return nil, fmt.Errorf("writing network configuration script: %w", err)
	}

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
	combustionNetworkDir := filepath.Join(ctx.CombustionDir, networkConfigDir, nmcConfigDir)

	cmd := exec.Command(nmcExecutable, "generate",
		"--config-dir", configDir,
		"--output-dir", combustionNetworkDir)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("running nmc generate: %w", err)
	}

	return nil
}

func generateNetworkLogFilename(ctx *image.Context) string {
	const networkConfigLogFile = "network-config-%s.log"

	timestamp := time.Now().Format("Jan02_15-04-05")
	filename := fmt.Sprintf(networkConfigLogFile, timestamp)

	return filepath.Join(ctx.BuildDir, filename)
}

func writeNMCExecutable(ctx *image.Context) error {
	nmcPath, err := exec.LookPath(nmcExecutable)
	if err != nil {
		return fmt.Errorf("searching for executable: %w", err)
	}

	destPath := filepath.Join(ctx.CombustionDir, networkConfigDir, nmcExecutable)
	if err = fileio.CopyFile(nmcPath, destPath, fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("copying executable: %w", err)
	}

	return nil
}

func writeNetworkConfigurationScript(ctx *image.Context) (string, error) {
	values := struct {
		NMCExecutablePath string
		ConfigDir         string
	}{
		ConfigDir:         filepath.Join(networkConfigDir, nmcConfigDir),
		NMCExecutablePath: filepath.Join(networkConfigDir, nmcExecutable),
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
