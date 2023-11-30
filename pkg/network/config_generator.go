package network

import (
	"fmt"
	"io"
	"os/exec"
)

type ConfigGenerator struct{}

func (ConfigGenerator) GenerateNetworkConfig(configDir, outputDir string, outputWriter io.Writer) error {
	cmd := generateCommand(configDir, outputDir, outputWriter)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running generate command: %w", err)
	}

	return nil
}

func generateCommand(configDir, outputDir string, output io.Writer) *exec.Cmd {
	cmd := exec.Command("nmc", "generate",
		"--config-dir", configDir,
		"--output-dir", outputDir)

	cmd.Stdout = output
	cmd.Stderr = output

	return cmd
}
