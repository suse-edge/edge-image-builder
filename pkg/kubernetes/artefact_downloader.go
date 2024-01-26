package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
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

type cache interface {
	Get(artefact string) (filepath string, err error)
	Put(artefact string, reader io.Reader) error
}

type ArtefactDownloader struct {
	Cache cache
}

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

	if err = d.downloadArtefacts(artefacts, rke2ReleaseURL, version, imagesDestination); err != nil {
		return "", "", fmt.Errorf("downloading RKE2 image artefacts: %w", err)
	}

	artefacts = installerArtefacts(arch)
	if err = d.downloadArtefacts(artefacts, rke2ReleaseURL, version, installDestination); err != nil {
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

func (d ArtefactDownloader) downloadArtefacts(artefacts []string, releaseURL, version, destinationPath string) error {
	for _, artefact := range artefacts {
		url := fmt.Sprintf(releaseURL, version, artefact)
		path := filepath.Join(destinationPath, artefact)

		copied, err := d.copyArtefactFromCache(artefact, path)
		if err != nil {
			return fmt.Errorf("retrieving artefact '%s' from cache: %w", artefact, err)
		}

		if !copied {
			if err = d.downloadArtefact(url, path, artefact); err != nil {
				return fmt.Errorf("downloading artefact '%s': %w", artefact, err)
			}
		}
	}

	return nil
}

func (d ArtefactDownloader) copyArtefactFromCache(artefact, destPath string) (bool, error) {
	sourcePath, err := d.Cache.Get(artefact)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("querying cache: %w", err)
	}

	zap.S().Infof("Copying artefact '%s' from cache", artefact)

	if err = fileio.CopyFile(sourcePath, destPath, fileio.NonExecutablePerms); err != nil {
		return false, fmt.Errorf("copying from cache: %w", err)
	}

	return true, nil
}

func (d ArtefactDownloader) downloadArtefact(url, path, artefact string) error {
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
		if err := d.Cache.Put(artefact, reader); err != nil {
			return fmt.Errorf("caching artefact: %w", err)
		}

		return nil
	})

	return errGroup.Wait()
}
