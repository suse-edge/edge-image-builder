package image

import (
	"bytes"
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
	KernelArgs       []string               `yaml:"kernelArgs"`
	Groups           []OperatingSystemGroup `yaml:"groups"`
	Users            []OperatingSystemUser  `yaml:"users"`
	Systemd          Systemd                `yaml:"systemd"`
	Suma             Suma                   `yaml:"suma"`
	Packages         Packages               `yaml:"packages"`
	IsoConfiguration IsoConfiguration       `yaml:"isoConfiguration"`
	RawConfiguration RawConfiguration       `yaml:"rawConfiguration"`
	Time             Time                   `yaml:"time"`
	Proxy            Proxy                  `yaml:"proxy"`
	Keymap           string                 `yaml:"keymap"`
}

type IsoConfiguration struct {
	InstallDevice string `yaml:"installDevice"`
	Unattended    bool   `yaml:"unattended"`
}

type RawConfiguration struct {
	DiskSize string `yaml:"diskSize"`
}

type Packages struct {
	NoGPGCheck      bool      `yaml:"noGPGCheck"`
	PKGList         []string  `yaml:"packageList"`
	AdditionalRepos []AddRepo `yaml:"additionalRepos"`
	RegCode         string    `yaml:"sccRegistrationCode"`
}

type AddRepo struct {
	URL      string `yaml:"url"`
	Unsigned bool   `yaml:"unsigned"`
}

type OperatingSystemUser struct {
	Username          string   `yaml:"username"`
	UID               int      `yaml:"uid"`
	EncryptedPassword string   `yaml:"encryptedPassword"`
	SSHKeys           []string `yaml:"sshKeys"`
	PrimaryGroup      string   `yaml:"primaryGroup"`
	SecondaryGroups   []string `yaml:"secondaryGroups"`
	CreateHome        bool     `yaml:"createHome"`
}

type OperatingSystemGroup struct {
	Name string `yaml:"name"`
	GID  int    `yaml:"gid"`
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
	ForceWait bool     `yaml:"forceWait"`
	Pools     []string `yaml:"pools"`
	Servers   []string `yaml:"servers"`
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
	Name string `yaml:"name"`
}

type Kubernetes struct {
	Version   string    `yaml:"version"`
	Network   Network   `yaml:"network"`
	Nodes     []Node    `yaml:"nodes"`
	Manifests Manifests `yaml:"manifests"`
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

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(&definition); err != nil {
		return nil, fmt.Errorf("could not parse the image definition: %w", err)
	}
	definition.Image.ImageType = strings.ToLower(definition.Image.ImageType)

	return &definition, nil
}
