package image

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	TypeISO = "iso"
	TypeRAW = "raw"
)

type Definition struct {
	APIVersion               string                   `yaml:"apiVersion"`
	Image                    Image                    `yaml:"image"`
	OperatingSystem          OperatingSystem          `yaml:"operatingSystem"`
	EmbeddedArtifactRegistry EmbeddedArtifactRegistry `yaml:"embeddedArtifactRegistry"`
}

type Image struct {
	ImageType       string `yaml:"imageType"`
	BaseImage       string `yaml:"baseImage"`
	OutputImageName string `yaml:"outputImageName"`
}

type OperatingSystem struct {
	KernelArgs []string              `yaml:"kernelArgs"`
	Users      []OperatingSystemUser `yaml:"users"`
	Systemd    Systemd               `yaml:"systemd"`
	Suma       Suma                  `yaml:"suma"`
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
	GetSSL        bool   `yaml:"getSSL"`
}

type EmbeddedArtifactRegistry struct {
	ContainerImages []ContainerImage `yaml:"images"`
	HelmCharts      []HelmChart      `yaml:"charts"`
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

func ParseDefinition(data []byte) (*Definition, error) {
	var definition Definition

	if err := yaml.Unmarshal(data, &definition); err != nil {
		return nil, fmt.Errorf("could not parse the image definition: %w", err)
	}
	definition.Image.ImageType = strings.ToLower(definition.Image.ImageType)

	return &definition, nil
}
