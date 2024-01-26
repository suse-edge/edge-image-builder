package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/http"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	rke2ReleaseURL = "https://github.com/rancher/rke2/releases/download/%s/%s"
	k3sReleaseURL  = "https://github.com/k3s-io/k3s/releases/download/%s/%s"

	rke2Binary     = "rke2.linux-%s.tar.gz"
	rke2CoreImages = "rke2-images-core.linux-%s.tar.zst"
	rke2Checksums  = "sha256sum-%s.txt"

	rke2CalicoImages = "rke2-images-calico.linux-%s.tar.zst"
	rke2CanalImages  = "rke2-images-canal.linux-%s.tar.zst"
	rke2CiliumImages = "rke2-images-cilium.linux-%s.tar.zst"
	rke2MultusImages = "rke2-images-multus.linux-%s.tar.zst"

	k3sBinary = "k3s"
	k3sImages = "k3s-airgap-images-%s.tar.zst"
)

type cache interface {
	Get(artefact string) (filepath string, err error)
	Put(artefact string, reader io.Reader) error
}

type ArtefactDownloader struct {
	Cache cache
}

func (d ArtefactDownloader) DownloadRKE2Artefacts(arch image.Arch, version, cni string, multusEnabled bool, installPath, imagesPath string) error {
	if !strings.Contains(version, image.KubernetesDistroRKE2) {
		return fmt.Errorf("invalid RKE2 version: '%s'", version)
	}

	if arch == image.ArchTypeARM {
		log.Audit("WARNING: RKE2 support for aarch64 platforms is limited and experimental")
	}

	artefacts, err := rke2ImageArtefacts(cni, multusEnabled, arch)
	if err != nil {
		return fmt.Errorf("gathering RKE2 image artefacts: %w", err)
	}

	if err = d.downloadArtefacts(artefacts, rke2ReleaseURL, version, imagesPath); err != nil {
		return fmt.Errorf("downloading RKE2 image artefacts: %w", err)
	}

	artefacts = rke2InstallerArtefacts(arch)
	if err = d.downloadArtefacts(artefacts, rke2ReleaseURL, version, installPath); err != nil {
		return fmt.Errorf("downloading RKE2 install artefacts: %w", err)
	}

	return nil
}

func rke2InstallerArtefacts(arch image.Arch) []string {
	artefactArch := arch.Short()

	return []string{
		fmt.Sprintf(rke2Binary, artefactArch),
		fmt.Sprintf(rke2Checksums, artefactArch),
	}
}

func rke2ImageArtefacts(cni string, multusEnabled bool, arch image.Arch) ([]string, error) {
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

func (d ArtefactDownloader) DownloadK3sArtefacts(arch image.Arch, version, installPath, imagesPath string) error {
	if !strings.Contains(version, image.KubernetesDistroK3S) {
		return fmt.Errorf("invalid k3s version: '%s'", version)
	}

	artefacts := k3sImageArtefacts(arch)
	if err := d.downloadArtefacts(artefacts, k3sReleaseURL, version, imagesPath); err != nil {
		return fmt.Errorf("downloading k3s image artefacts: %w", err)
	}

	artefacts = k3sInstallerArtefacts(arch)
	if err := d.downloadArtefacts(artefacts, k3sReleaseURL, version, installPath); err != nil {
		return fmt.Errorf("downloading k3s install artefacts: %w", err)
	}

	return nil
}

func k3sInstallerArtefacts(arch image.Arch) []string {
	artefactArch := arch.Short()

	binary := k3sBinary
	if arch == image.ArchTypeARM {
		binary = fmt.Sprintf("%s-%s", k3sBinary, artefactArch)
	}

	return []string{
		binary,
	}
}

func k3sImageArtefacts(arch image.Arch) []string {
	artefactArch := arch.Short()

	return []string{
		fmt.Sprintf(k3sImages, artefactArch),
	}
}

func (d ArtefactDownloader) downloadArtefacts(artefacts []string, releaseURL, version, destinationPath string) error {
	for _, artefact := range artefacts {
		url := fmt.Sprintf(releaseURL, version, artefact)
		path := filepath.Join(destinationPath, artefact)
		cacheKey := cacheIdentifier(version, artefact)

		copied, err := d.copyArtefactFromCache(cacheKey, path)
		if err != nil {
			return fmt.Errorf("retrieving artefact '%s' from cache: %w", artefact, err)
		}

		if !copied {
			if err = d.downloadArtefact(url, path, cacheKey); err != nil {
				return fmt.Errorf("downloading artefact '%s': %w", artefact, err)
			}
		}
	}

	return nil
}

func (d ArtefactDownloader) copyArtefactFromCache(cacheKey, destPath string) (bool, error) {
	sourcePath, err := d.Cache.Get(cacheKey)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("querying cache: %w", err)
	}

	zap.S().Infof("Copying artefact with identifier '%s' from cache", cacheKey)

	if err = fileio.CopyFile(sourcePath, destPath, fileio.NonExecutablePerms); err != nil {
		return false, fmt.Errorf("copying from cache: %w", err)
	}

	return true, nil
}

func (d ArtefactDownloader) downloadArtefact(url, path, cacheKey string) error {
	reader, writer := io.Pipe()

	errGroup, ctx := errgroup.WithContext(context.Background())

	errGroup.Go(func() error {
		defer func() {
			if err := writer.Close(); err != nil {
				zap.S().Warnf("Closing pipe writer failed unexpectedly: %v", err)
			}
		}()

		if err := http.DownloadFile(ctx, url, path, writer); err != nil {
			return fmt.Errorf("downloading artefact: %w", err)
		}
		return nil
	})

	errGroup.Go(func() error {
		if err := d.Cache.Put(cacheKey, reader); err != nil {
			return fmt.Errorf("caching artefact: %w", err)
		}

		return nil
	})

	return errGroup.Wait()
}

func cacheIdentifier(version, artefact string) string {
	return fmt.Sprintf("%s/%s", version, artefact)
}
