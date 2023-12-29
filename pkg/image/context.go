package image

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type NetworkConfigGenerator interface {
	GenerateNetworkConfig(configDir, outputDir string, outputWriter io.Writer) error
}

type NetworkConfiguratorInstaller interface {
	InstallConfigurator(imageName, sourcePath, installPath string) error
}

type Context struct {
	// ImageConfigDir is the root directory storing all configuration files.
	ImageConfigDir string
	// BuildDir is the directory used for assembling the different components used in a build.
	BuildDir string
	// CombustionDir is a subdirectory under BuildDir containing the Combustion script and all related files.
	CombustionDir string
	// DeleteBuildDir indicates whether the BuildDir should be cleaned up after the image is built.
	DeleteBuildDir bool
	// ImageDefinition contains the image definition properties.
	ImageDefinition              *Definition
	NetworkConfigGenerator       NetworkConfigGenerator
	NetworkConfiguratorInstaller NetworkConfiguratorInstaller
}

func NewContext(
	imageConfigDir string,
	rootBuildDir string,
	deleteBuildDir bool,
	definition *Definition,
	generator NetworkConfigGenerator,
	installer NetworkConfiguratorInstaller,
) (*Context, error) {
	if rootBuildDir == "" {
		tmpDir, err := os.MkdirTemp("", "eib-")
		if err != nil {
			return nil, fmt.Errorf("creating a temporary build directory: %w", err)
		}
		rootBuildDir = tmpDir
	}

	timestamp := time.Now().Format("Jan02_15-04-05")
	buildDir := filepath.Join(rootBuildDir, fmt.Sprintf("build-%s", timestamp))

	if err := os.Mkdir(buildDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating a build directory: %w", err)
	}

	combustionDir := filepath.Join(buildDir, "combustion")

	if err := os.Mkdir(combustionDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating the combustion directory: %w", err)
	}

	return &Context{
		ImageConfigDir:               imageConfigDir,
		BuildDir:                     buildDir,
		CombustionDir:                combustionDir,
		DeleteBuildDir:               deleteBuildDir,
		ImageDefinition:              definition,
		NetworkConfigGenerator:       generator,
		NetworkConfiguratorInstaller: installer,
	}, nil
}

func CleanUpBuildDir(c *Context) error {
	if c.DeleteBuildDir {
		err := os.RemoveAll(c.BuildDir)
		if err != nil {
			return fmt.Errorf("deleting build directory: %w", err)
		}
	}
	return nil
}
