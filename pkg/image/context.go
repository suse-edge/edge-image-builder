package image

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
	// ArtifactSources contains the information necessary for the deployment of external artifacts.
	ArtifactSources *ArtifactSources
	// CacheDir contains all of the artifacts that are cached for the build process.
	CacheDir string
}

type ArtifactSources struct {
	MetalLB struct {
		Chart      string `yaml:"chart"`
		Repository string `yaml:"repository"`
		Version    string `yaml:"version"`
	} `yaml:"metallb"`
	EndpointCopierOperator struct {
		Chart      string `yaml:"chart"`
		Repository string `yaml:"repository"`
		Version    string `yaml:"version"`
	} `yaml:"endpoint-copier-operator"`
	Kubernetes struct {
		K3s struct {
			SELinuxPackage    string `yaml:"selinuxPackage"`
			SELinuxRepository string `yaml:"selinuxRepository"`
		} `yaml:"k3s"`
		Rke2 struct {
			SELinuxPackage    string `yaml:"selinuxPackage"`
			SELinuxRepository string `yaml:"selinuxRepository"`
		} `yaml:"rke2"`
	} `yaml:"kubernetes"`
}
