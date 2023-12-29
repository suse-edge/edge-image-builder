package image

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type NetworkConfigGenerator interface {
	GenerateNetworkConfig(configDir, outputDir string, outputWriter io.Writer) error
}

type NetworkConfiguratorInstaller interface {
	InstallConfigurator(imageName, sourcePath, installPath string) error
}

type Context struct {
	// ImageConfigDir is the root directory storing all configuration files.
	ImageConfigDir string
	// BuildDir is the directory used for assembling the different components used in a build.
	BuildDir string
	// CombustionDir is a subdirectory under BuildDir containing the Combustion script and all related files.
	CombustionDir string
	// ImageDefinition contains the image definition properties.
	ImageDefinition              *Definition
	NetworkConfigGenerator       NetworkConfigGenerator
	NetworkConfiguratorInstaller NetworkConfiguratorInstaller
}

func NewContext(
	imageConfigDir string,
	rootBuildDir string,
	definition *Definition,
	generator NetworkConfigGenerator,
	installer NetworkConfiguratorInstaller,
) (*Context, error) {
	if rootBuildDir == "" {
		tmpDir, err := os.MkdirTemp("", "eib-")
		if err != nil {
			return nil, fmt.Errorf("creating a temporary build directory: %w", err)
		}
		rootBuildDir = tmpDir
	}

	timestamp := time.Now().Format("Jan02_15-04-05")
	buildDir := filepath.Join(rootBuildDir, fmt.Sprintf("build-%s", timestamp))

	if err := os.Mkdir(buildDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating a build directory: %w", err)
	}

	combustionDir := filepath.Join(buildDir, "combustion")

	if err := os.Mkdir(combustionDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating the combustion directory: %w", err)
	}

	return &Context{
		ImageConfigDir:               imageConfigDir,
		BuildDir:                     buildDir,
		CombustionDir:                combustionDir,
		ImageDefinition:              definition,
		NetworkConfigGenerator:       generator,
		NetworkConfiguratorInstaller: installer,
	}, nil
}
