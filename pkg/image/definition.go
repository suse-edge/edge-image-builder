package image

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
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

var (
	diskSizeRegexp = regexp.MustCompile(`^([1-9]\d+|[1-9])+([MGT])`)
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
}

type DiskSize string

func (d DiskSize) IsValid() bool {
	return diskSizeRegexp.MatchString(string(d))
}

func (d DiskSize) ToMB() int64 {
	if d == "" {
		return 0
	}

	s := diskSizeRegexp.FindStringSubmatch(string(d))
	if len(s) != 3 {
		panic("unknown disk size format")
	}

	quantity, err := strconv.Atoi(s[1])
	if err != nil {
		panic(fmt.Sprintf("invalid disk size: %s", string(d)))
	}

	sizeType := s[2]

	switch sizeType {
	case "M":
		return int64(quantity)
	case "G":
		return int64(quantity) * 1024
	case "T":
		return int64(quantity) * 1024 * 1024
	default:
		panic("unknown disk size type")
	}
}

type RawConfiguration struct {
	DiskSize DiskSize `yaml:"diskSize"`
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
	CreateHomeDir     bool     `yaml:"createHomeDir"`
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
	Helm      Helm      `yaml:"helm"`
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

type Helm struct {
	Charts       []HelmChart      `yaml:"charts"`
	Repositories []HelmRepository `yaml:"repositories"`
}

type HelmChart struct {
	Name                  string `yaml:"name"`
	RepositoryName        string `yaml:"repositoryName"`
	Version               string `yaml:"version"`
	TargetNamespace       string `yaml:"targetNamespace"`
	CreateNamespace       bool   `yaml:"createNamespace"`
	InstallationNamespace string `yaml:"installationNamespace"`
	ValuesFile            string `yaml:"valuesFile"`
}

type HelmRepository struct {
	Name           string             `yaml:"name"`
	URL            string             `yaml:"url"`
	Authentication HelmAuthentication `yaml:"authentication"`
	PlainHTTP      bool               `yaml:"plainHTTP"`
	SkipTLSVerify  bool               `yaml:"skipTLSVerify"`
	CAFile         string             `yaml:"caFile"`
}

type HelmAuthentication struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
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
