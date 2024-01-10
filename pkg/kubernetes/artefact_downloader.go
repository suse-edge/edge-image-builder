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
	"golang.org/x/sync/errgroup"
)

const (
	kubernetesDir = "kubernetes"

	rke2ReleaseURL = "https://github.com/rancher/rke2/releases/download/%s/%s"

	rke2Binary     = "rke2.linux-%s.tar.gz"
	rke2CoreImages = "rke2-images-core.linux-%s.tar.zst"
	rke2Checksums  = "sha256sum-%s.txt"

	rke2CalicoImages = "rke2-images-calico.linux-%s.tar.zst"
	rke2CanalImages  = "rke2-images-canal.linux-%s.tar.zst"
	rke2CiliumImages = "rke2-images-cilium.linux-%s.tar.zst"
	rke2MultusImages = "rke2-images-multus.linux-%s.tar.zst"

	rke2VSphereImages = "rke2-images-vsphere.linux-%s.tar.zst"
)

type ArtefactDownloader struct{}

func (d ArtefactDownloader) DownloadArtefacts(kubernetes image.Kubernetes, arch image.Arch, destinationPath string) error {
	if !strings.Contains(kubernetes.Version, image.KubernetesDistroRKE2) {
		return fmt.Errorf("kubernetes version '%s' is not supported", kubernetes.Version)
	}

	path := filepath.Join(destinationPath, kubernetesDir)

	if err := os.Mkdir(path, os.ModePerm); err != nil {
		return fmt.Errorf("creating kubernetes dir: %w", err)
	}

	artefacts, err := gatherArtefacts(kubernetes, arch)
	if err != nil {
		return fmt.Errorf("gathering RKE2 artefacts: %w", err)
	}

	if err = downloadArtefacts(artefacts, rke2ReleaseURL, kubernetes.Version, destinationPath); err != nil {
		return fmt.Errorf("downloading RKE2 artefacts: %w", err)
	}

	return nil
}

func gatherArtefacts(kubernetes image.Kubernetes, arch image.Arch) ([]string, error) {
	if arch == image.ArchTypeARM {
		log.Audit("WARNING: RKE2 support for aarch64 platforms is limited and experimental")
	}

	artefactArch := arch.Short()

	var artefacts []string

	artefacts = append(artefacts,
		fmt.Sprintf(rke2Binary, artefactArch),
		fmt.Sprintf(rke2CoreImages, artefactArch),
		fmt.Sprintf(rke2Checksums, artefactArch))

	switch kubernetes.CNI {
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
		return nil, fmt.Errorf("unsupported CNI: %s", kubernetes.CNI)
	}

	if kubernetes.MultusEnabled {
		if arch == image.ArchTypeARM {
			return nil, fmt.Errorf("multus is not supported on %s platforms", arch)
		}
		artefacts = append(artefacts, fmt.Sprintf(rke2MultusImages, artefactArch))
	}

	if kubernetes.VSphereEnabled {
		if arch == image.ArchTypeARM {
			return nil, fmt.Errorf("vSphere is not supported on %s platforms", arch)
		}
		artefacts = append(artefacts, fmt.Sprintf(rke2VSphereImages, artefactArch))
	}

	return artefacts, nil
}

func downloadArtefacts(artefacts []string, releaseURL, version, destinationPath string) error {
	errGroup, ctx := errgroup.WithContext(context.Background())

	for _, artefact := range artefacts {
		func(artefact string) {
			errGroup.Go(func() error {
				url := fmt.Sprintf(releaseURL, version, artefact)
				path := filepath.Join(destinationPath, artefact)

				if err := http.DownloadFile(ctx, url, path); err != nil {
					return fmt.Errorf("downloading artefact '%s': %w", artefact, err)
				}

				return nil
			})
		}(artefact)
	}

	return errGroup.Wait()
}
