package registry

import (
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type helmClient interface {
	AddRepo(repository *image.HelmRepository) error
	RegistryLogin(repository *image.HelmRepository) error
	Pull(chart string, repository *image.HelmRepository, version, destDir string) (string, error)
	Template(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error)
}

type Registry struct {
	HelmClient helmClient
}
