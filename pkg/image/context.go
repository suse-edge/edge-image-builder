package image

import (
	"io"
)

type networkConfigGenerator interface {
	GenerateNetworkConfig(configDir, outputDir string, outputWriter io.Writer) error
}

type networkConfiguratorInstaller interface {
	InstallConfigurator(arch Arch, sourcePath, installPath string) error
}

type kubernetesScriptInstaller interface {
	InstallScript(distribution, sourcePath, destinationPath string) error
}

type kubernetesArtefactDownloader interface {
	DownloadArtefacts(kubernetes Kubernetes, arch Arch, destinationPath string) (string, string, error)
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
	NetworkConfigGenerator       networkConfigGenerator
	NetworkConfiguratorInstaller networkConfiguratorInstaller
	KubernetesScriptInstaller    kubernetesScriptInstaller
	KubernetesArtefactDownloader kubernetesArtefactDownloader
}
