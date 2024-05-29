package image

type HelmClient interface {
	AddRepo(repository *HelmRepository) error
	RegistryLogin(repository *HelmRepository) error
	Pull(chart string, repository *HelmRepository, version, destDir string) (string, error)
	Template(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error)
}

type LocalRPMConfig struct {
	// RPMPath is the path to the directory holding RPMs that will be side-loaded
	RPMPath string
	// GPGKeysPath specifies the path to the directory that holds the GPG keys that the side-loaded RPMs have been signed with
	GPGKeysPath string
}

type Context struct {
	// ImageConfigDir is the root directory storing all configuration files.
	ImageConfigDir string
	// BuildDir is the directory used for assembling the different components used in a build.
	BuildDir string
	// CombustionDir is a subdirectory under BuildDir containing the Combustion script and its smaller related files.
	CombustionDir string
	// ArtefactsDir is a subdirectory under BuildDir containing the larger Combustion related files.
	ArtefactsDir string
	// ImageDefinition contains the image definition properties.
	ImageDefinition *Definition
}
