package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/http"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func SELinuxPackage(version string) string {
	const (
		k3sPackage  = "k3s-selinux"
		rke2Package = "rke2-selinux"
	)

	switch {
	case strings.Contains(version, image.KubernetesDistroK3S):
		return k3sPackage
	case strings.Contains(version, image.KubernetesDistroRKE2):
		return rke2Package
	default:
		message := fmt.Sprintf("invalid kubernetes version: %s", version)
		panic(message)
	}
}

func SELinuxRepository(version string) image.AddRepo {
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
		message := fmt.Sprintf("invalid kubernetes version: %s", version)
		panic(message)
	}

	return image.AddRepo{
		URL:      url,
		Unsigned: true,
	}
}

func DownloadSELinuxRPMsSigningKey(gpgKeysDir string) error {
	const rancherSigningKeyURL = "https://rpm.rancher.io/public.key"
	var signingKeyPath = filepath.Join(gpgKeysDir, "rancher-public.key")

	return http.DownloadFile(context.Background(), rancherSigningKeyURL, signingKeyPath, nil)
}
