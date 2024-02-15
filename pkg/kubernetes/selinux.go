package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/http"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func SELinuxPackage(version string) (string, error) {
	const (
		k3sPackage  = "k3s-selinux"
		rke2Package = "rke2-selinux"
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

func SELinuxRepository(version string) (image.AddRepo, error) {
	const (
		k3sRepository  = "https://rpm.rancher.io/k3s/stable/common/slemicro/noarch"
		rke2Repository = "https://rpm.rancher.io/rke2/stable/common/slemicro/noarch"
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
