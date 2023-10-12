package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type ImageConfig struct {
	APIVersion string `yaml:"apiVersion"`
	Image      struct {
		ImageType       string `yaml:"imageTygipe"`
		BaseImage       string `yaml:"baseImage"`
		OutputImageName string `yaml:"outputImageName"`
	}
}

func Parse(data []byte) (*ImageConfig, error) {
	imageConfig := ImageConfig{}

	err := yaml.Unmarshal(data, &imageConfig)
	if err != nil {
		return nil, fmt.Errorf("could not parse the image configuration: %w", err)
	}

	return &imageConfig, nil
}
