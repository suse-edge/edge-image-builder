package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type ImageConfig struct {
	// EIB Metadata
	ApiVersion string `yaml:"apiVersion"`

	// Image Base Configuration
	ImageType  string `yaml:"imageType"`
}

func Parse(data []byte) (*ImageConfig, error) {
	imageConfig := ImageConfig{}

	err := yaml.Unmarshal(data, &imageConfig)
	if err != nil {
		return nil, fmt.Errorf("could not parse the image configuration: %w", err)
	}

	return &imageConfig, nil
}
