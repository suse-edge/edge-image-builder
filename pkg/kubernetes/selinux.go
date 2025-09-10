package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/http"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func SELinuxPackage(version string, sources *image.ArtifactSources) (string, error) {

	switch {
	case strings.Contains(version, image.KubernetesDistroK3S):
		return sources.Kubernetes.K3s.SELinuxPackage, nil
	case strings.Contains(version, image.KubernetesDistroRKE2):
		return sources.Kubernetes.Rke2.SELinuxPackage, nil
	default:
		return "", fmt.Errorf("invalid kubernetes version: %s", version)
	}
}

func SELinuxRepository(version string, sources *image.ArtifactSources) (image.AddRepo, error) {
	var url string

	switch {
	case strings.Contains(version, image.KubernetesDistroK3S):
		url = sources.Kubernetes.K3s.SELinuxRepository
	case strings.Contains(version, image.KubernetesDistroRKE2):
		url = sources.Kubernetes.Rke2.SELinuxRepository
	default:
		return image.AddRepo{}, fmt.Errorf("invalid kubernetes version: %s", version)
	}

	return image.AddRepo{
		URL:      url,
		Unsigned: true,
	}, nil
}

func SELinuxRepositoryPriority(version string, sources *image.ArtifactSources) (int, error) {
	var priority int

	switch {
	case strings.Contains(version, image.KubernetesDistroK3S):
		priority = sources.Kubernetes.K3s.SELinuxRepositoryPriority
	case strings.Contains(version, image.KubernetesDistroRKE2):
		priority = sources.Kubernetes.Rke2.SELinuxRepositoryPriority
	default:
		return 0, fmt.Errorf("invalid kubernetes version: %s", version)
	}

	return priority, nil
}

func DownloadSELinuxRPMsSigningKey(gpgKeysDir string) error {
	const rancherSigningKeyURL = "https://rpm.rancher.io/public.key"
	var signingKeyPath = filepath.Join(gpgKeysDir, "rancher-public.key")

	return http.DownloadFile(context.Background(), rancherSigningKeyURL, signingKeyPath, nil)
}
