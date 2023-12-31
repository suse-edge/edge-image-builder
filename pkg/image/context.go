package image

import (
	"io"
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
