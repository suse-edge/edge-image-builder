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
	var (
		k3sPackage  = sources.Kubernetes.K3s.SelinuxPackage
		rke2Package = sources.Kubernetes.Rke2.SelinuxPackage
	)

	switch {
	case strings.Contains(version, image.KubernetesDistroK3S):
		return k3sPackage, nil
	case strings.Contains(version, image.KubernetesDistroRKE2):
		return rke2Package, nil
	default:
		return "", fmt.Errorf("invalid kubernetes version: %s", version)
	}
}

func SELinuxRepository(version string, sources *image.ArtifactSources) (image.AddRepo, error) {
	var (
		k3sRepository  = sources.Kubernetes.K3s.SelinuxRepository
		rke2Repository = sources.Kubernetes.Rke2.SelinuxRepository
	)

	var url string

	switch {
	case strings.Contains(version, image.KubernetesDistroK3S):
		url = k3sRepository
	case strings.Contains(version, image.KubernetesDistroRKE2):
		url = rke2Repository
	default:
		return image.AddRepo{}, fmt.Errorf("invalid kubernetes version: %s", version)
	}

	return image.AddRepo{
		URL:      url,
		Unsigned: true,
	}, nil
}

func DownloadSELinuxRPMsSigningKey(gpgKeysDir string) error {
	const rancherSigningKeyURL = "https://rpm.rancher.io/public.key"
	var signingKeyPath = filepath.Join(gpgKeysDir, "rancher-public.key")

	return http.DownloadFile(context.Background(), rancherSigningKeyURL, signingKeyPath, nil)
}
