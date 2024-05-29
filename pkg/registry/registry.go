package registry

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"go.uber.org/zap"
)

type helmClient interface {
	AddRepo(repository *image.HelmRepository) error
	RegistryLogin(repository *image.HelmRepository) error
	Pull(chart string, repository *image.HelmRepository, version, destDir string) (string, error)
	Template(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error)
}

type Registry struct {
	helmClient   helmClient
	manifestsDir string
}

func New(ctx *image.Context, helmClient helmClient, localManifestsDir string) (*Registry, error) {
	manifestsDir, err := storeManifests(ctx, localManifestsDir)
	if err != nil {
		return nil, fmt.Errorf("storing manifests: %w", err)
	}

	return &Registry{
		helmClient:   helmClient,
		manifestsDir: manifestsDir,
	}, nil
}

func (r *Registry) ManifestsPath() string {
	return r.manifestsDir
}

func storeManifests(ctx *image.Context, localManifestsDir string) (string, error) {
	manifestsDestDir := filepath.Join(ctx.BuildDir, "manifests")

	manifestURLs := ctx.ImageDefinition.Kubernetes.Manifests.URLs
	if len(manifestURLs) != 0 {
		if err := os.MkdirAll(manifestsDestDir, os.ModePerm); err != nil {
			return "", fmt.Errorf("creating manifests dir: %w", err)
		}

		_, err := downloadManifests(manifestURLs, manifestsDestDir)
		if err != nil {
			return "", fmt.Errorf("downloading manifests: %w", err)
		}
	}

	if err := copyLocalManifests(localManifestsDir, manifestsDestDir); err != nil {
		return "", fmt.Errorf("copying local manifests: %w", err)
	}

	return manifestsDestDir, nil
}

func copyLocalManifests(localManifestsDir, manifestsBuildDir string) error {
	if _, err := os.Stat(localManifestsDir); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.S().Warnf("Searching for local manifests failed: %v", err)
		}

		return nil
	}

	if err := fileio.CopyFiles(localManifestsDir, manifestsBuildDir, ".yaml", false); err != nil {
		return fmt.Errorf("copying manifests: %w", err)
	}
	if err := fileio.CopyFiles(localManifestsDir, manifestsBuildDir, ".yml", false); err != nil {
		return fmt.Errorf("copying manifests: %w", err)
	}

	return nil
}
