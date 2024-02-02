package image

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	TypeISO = "iso"
	TypeRAW = "raw"

	ArchTypeX86 Arch = "x86_64"
	ArchTypeARM Arch = "aarch64"

	KubernetesDistroRKE2 = "rke2"
	KubernetesDistroK3S  = "k3s"

	KubernetesNodeTypeServer = "server"
	KubernetesNodeTypeAgent  = "agent"

	CNITypeNone   = "none"
	CNITypeCilium = "cilium"
	CNITypeCanal  = "canal"
	CNITypeCalico = "calico"
)

type Definition struct {
	APIVersion               string                   `yaml:"apiVersion"`
	Image                    Image                    `yaml:"image"`
	OperatingSystem          OperatingSystem          `yaml:"operatingSystem"`
	EmbeddedArtifactRegistry EmbeddedArtifactRegistry `yaml:"embeddedArtifactRegistry"`
	Kubernetes               Kubernetes               `yaml:"kubernetes"`
}

type Arch string

func (a Arch) Short() string {
	switch a {
	case ArchTypeX86:
		return "amd64"
	case ArchTypeARM:
		return "arm64"
	default:
		message := fmt.Sprintf("unknown arch: %s", a)
		panic(message)
	}
}

type Image struct {
	ImageType       string `yaml:"imageType"`
	Arch            Arch   `yaml:"arch"`
	BaseImage       string `yaml:"baseImage"`
	OutputImageName string `yaml:"outputImageName"`
}

type OperatingSystem struct {
	KernelArgs      []string              `yaml:"kernelArgs"`
	Users           []OperatingSystemUser `yaml:"users"`
	Systemd         Systemd               `yaml:"systemd"`
	Suma            Suma                  `yaml:"suma"`
	Packages        Packages              `yaml:"packages"`
	IsoInstallation IsoInstallation       `yaml:"isoInstallation"`
	Time            Time                  `yaml:"time"`
	Proxy           Proxy                 `yaml:"proxy"`
	Keymap          string                `yaml:"keymap"`
}

type IsoInstallation struct {
	InstallDevice string `yaml:"installDevice"`
	Unattended    bool   `yaml:"unattended"`
}

type Packages struct {
	NoGPGCheck      bool      `yaml:"noGPGCheck"`
	PKGList         []string  `yaml:"packageList"`
	AdditionalRepos []AddRepo `yaml:"additionalRepos"`
	RegCode         string    `yaml:"sccRegistrationCode"`
}

type AddRepo struct {
	URL string `yaml:"url"`
}

type OperatingSystemUser struct {
	Username          string `yaml:"username"`
	EncryptedPassword string `yaml:"encryptedPassword"`
	SSHKey            string `yaml:"sshKey"`
}

type Systemd struct {
	Enable  []string `yaml:"enable"`
	Disable []string `yaml:"disable"`
}

type Suma struct {
	Host          string `yaml:"host"`
	ActivationKey string `yaml:"activationKey"`
}

type Time struct {
	Timezone         string           `yaml:"timezone"`
	NtpConfiguration NtpConfiguration `yaml:"ntp"`
}

type NtpConfiguration struct {
	Pools   []string `yaml:"pools"`
	Servers []string `yaml:"servers"`
}

type Proxy struct {
	HTTPProxy  string   `yaml:"httpProxy"`
	HTTPSProxy string   `yaml:"httpsProxy"`
	NoProxy    []string `yaml:"noProxy"`
}

type EmbeddedArtifactRegistry struct {
	ContainerImages []ContainerImage `yaml:"images"`
}

type ContainerImage struct {
	Name           string `yaml:"name"`
	SupplyChainKey string `yaml:"supplyChainKey"`
}

type HelmChart struct {
	Name    string `yaml:"name"`
	RepoURL string `yaml:"repoURL"`
	Version string `yaml:"version"`
}

type Kubernetes struct {
	Version    string      `yaml:"version"`
	Network    Network     `yaml:"network"`
	Nodes      []Node      `yaml:"nodes"`
	Manifests  Manifests   `yaml:"manifests"`
	HelmCharts []HelmChart `yaml:"charts"`
}

type Network struct {
	APIHost string `yaml:"apiHost"`
	APIVIP  string `yaml:"apiVIP"`
}

type Node struct {
	Hostname    string `yaml:"hostname"`
	Type        string `yaml:"type"`
	Initialiser bool   `yaml:"initializer"`
}

type Manifests struct {
	URLs []string `yaml:"urls"`
}

func ParseDefinition(data []byte) (*Definition, error) {
	var definition Definition

	if err := yaml.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("could not parse the image definition: %w", err)
	}
	definition.Image.ImageType = strings.ToLower(definition.Image.ImageType)

	return &definition, nil
}
