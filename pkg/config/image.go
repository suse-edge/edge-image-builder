package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	ImageTypeISO = "iso"
	ImageTypeRAW = "raw"
)

type ImageConfig struct {
	APIVersion      string          `yaml:"apiVersion"`
	Image           Image           `yaml:"image"`
	OperatingSystem OperatingSystem `yaml:"operatingSystem"`
}

type Image struct {
	ImageType       string `yaml:"imageType"`
	BaseImage       string `yaml:"baseImage"`
	OutputImageName string `yaml:"outputImageName"`
}

type OperatingSystem struct {
	KernelArgs []string              `yaml:"kernelArgs"`
	Users      []OperatingSystemUser `yaml:"users"`
}

type OperatingSystemUser struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSHKey   string `yaml:"sshKey"`
}

func Parse(data []byte) (*ImageConfig, error) {
	imageConfig := ImageConfig{}

	err := yaml.Unmarshal(data, &imageConfig)
	if err != nil {
		return nil, fmt.Errorf("could not parse the image configuration: %w", err)
	}

	return &imageConfig, nil
}
