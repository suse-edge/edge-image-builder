package build

import (
	"github.com/suse-edge/edge-image-builder/pkg/config"
	"os"
	"path/filepath"
)

func copyCombustionFile(sourcePath string, buildConfig *config.BuildConfig) error {
	src, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	destFilename := filepath.Join(buildConfig.CombustionDir, filepath.Base(sourcePath))
	err = os.WriteFile(destFilename, src, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}