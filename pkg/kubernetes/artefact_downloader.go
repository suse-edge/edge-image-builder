package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/http"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
)

const (
	kubernetesDir = "kubernetes"
	installDir    = "install"
	imagesDir     = "images"

	rke2ReleaseURL = "https://github.com/rancher/rke2/releases/download/%s/%s"

	rke2Binary     = "rke2.linux-%s.tar.gz"
	rke2CoreImages = "rke2-images-core.linux-%s.tar.zst"
	rke2Checksums  = "sha256sum-%s.txt"

	rke2CalicoImages = "rke2-images-calico.linux-%s.tar.zst"
	rke2CanalImages  = "rke2-images-canal.linux-%s.tar.zst"
	rke2CiliumImages = "rke2-images-cilium.linux-%s.tar.zst"
	rke2MultusImages = "rke2-images-multus.linux-%s.tar.zst"
)

type ArtefactDownloader struct{}

func (d ArtefactDownloader) DownloadArtefacts(arch image.Arch, version, cni string, multusEnabled bool, destinationPath string) (installPath, imagesPath string, err error) {
	if !strings.Contains(version, image.KubernetesDistroRKE2) {
		return "", "", fmt.Errorf("kubernetes version '%s' is not supported", version)
	}

	if arch == image.ArchTypeARM {
		log.Audit("WARNING: RKE2 support for aarch64 platforms is limited and experimental")
	}

	imagesPath = filepath.Join(kubernetesDir, imagesDir)
	imagesDestination := filepath.Join(destinationPath, imagesPath)
	if err = os.MkdirAll(imagesDestination, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating kubernetes images dir: %w", err)
	}

	installPath = filepath.Join(kubernetesDir, installDir)
	installDestination := filepath.Join(destinationPath, installPath)
	if err = os.MkdirAll(installDestination, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("creating kubernetes install dir: %w", err)
	}

	artefacts, err := imageArtefacts(cni, multusEnabled, arch)
	if err != nil {
		return "", "", fmt.Errorf("gathering RKE2 image artefacts: %w", err)
	}

	if err = downloadArtefacts(artefacts, rke2ReleaseURL, version, imagesDestination); err != nil {
		return "", "", fmt.Errorf("downloading RKE2 image artefacts: %w", err)
	}

	artefacts = installerArtefacts(arch)
	if err = downloadArtefacts(artefacts, rke2ReleaseURL, version, installDestination); err != nil {
		return "", "", fmt.Errorf("downloading RKE2 install artefacts: %w", err)
	}

	return installPath, imagesPath, nil
}

func installerArtefacts(arch image.Arch) []string {
	artefactArch := arch.Short()

	return []string{
		fmt.Sprintf(rke2Binary, artefactArch),
		fmt.Sprintf(rke2Checksums, artefactArch),
	}
}

func imageArtefacts(cni string, multusEnabled bool, arch image.Arch) ([]string, error) {
	artefactArch := arch.Short()

	var artefacts []string

	artefacts = append(artefacts, fmt.Sprintf(rke2CoreImages, artefactArch))

	switch cni {
	case "":
		return nil, fmt.Errorf("CNI not specified")
	case image.CNITypeNone:
	case image.CNITypeCanal:
		artefacts = append(artefacts, fmt.Sprintf(rke2CanalImages, artefactArch))
	case image.CNITypeCalico:
		if arch == image.ArchTypeARM {
			return nil, fmt.Errorf("calico is not supported on %s platforms", arch)
		}
		artefacts = append(artefacts, fmt.Sprintf(rke2CalicoImages, artefactArch))
	case image.CNITypeCilium:
		if arch == image.ArchTypeARM {
			return nil, fmt.Errorf("cilium is not supported on %s platforms", arch)
		}
		artefacts = append(artefacts, fmt.Sprintf(rke2CiliumImages, artefactArch))
	default:
		return nil, fmt.Errorf("unsupported CNI: %s", cni)
	}

	if multusEnabled {
		if arch == image.ArchTypeARM {
			return nil, fmt.Errorf("multus is not supported on %s platforms", arch)
		}
		artefacts = append(artefacts, fmt.Sprintf(rke2MultusImages, artefactArch))
	}

	return artefacts, nil
}

func downloadArtefacts(artefacts []string, releaseURL, version, destinationPath string) error {
	for _, artefact := range artefacts {
		url := fmt.Sprintf(releaseURL, version, artefact)
		path := filepath.Join(destinationPath, artefact)

		if err := http.DownloadFile(context.Background(), url, path, nil); err != nil {
			return fmt.Errorf("downloading artefact '%s': %w", artefact, err)
		}
	}

	return nil
}
