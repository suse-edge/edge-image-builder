package build

import (
	"fmt"
	"os"
	"path/filepath"
)

type Context struct {
	// ImageConfigDir is the root directory storing all configuration files.
	ImageConfigDir string
	// BuildDir is the directory used for assembling the different components used in a build.
	BuildDir string
	// CombustionDir is a subdirectory under BuildDir containing the Combustion script and all related files.
	CombustionDir string
	// DeleteBuildDir indicates whether the BuildDir should be cleaned up after the image is built.
	DeleteBuildDir bool
}

func NewContext(imageConfigDir, buildDir string, deleteBuildDir bool) (*Context, error) {
	if buildDir == "" {
		tmpDir, err := os.MkdirTemp("", "eib-")
		if err != nil {
			return nil, fmt.Errorf("creating a temporary build directory: %w", err)
		}
		buildDir = tmpDir
	}
	combustionDir := filepath.Join(buildDir, "combustion")

	if err := os.MkdirAll(combustionDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating the combustion directory: %w", err)
	}

	return &Context{
		ImageConfigDir: imageConfigDir,
		BuildDir:       buildDir,
		CombustionDir:  combustionDir,
		DeleteBuildDir: deleteBuildDir,
	}, nil
}

func (c *Context) CleanUpBuildDir() error {
	if c.DeleteBuildDir {
		err := os.RemoveAll(c.BuildDir)
		if err != nil {
			return fmt.Errorf("deleting build directory: %w", err)
		}
	}
	return nil
}
