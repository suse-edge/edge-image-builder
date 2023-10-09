package build

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/config"
)

func Build(imageConfig *config.ImageConfig, buildConfig *config.BuildConfig) error {
	err := prepareBuildDir(buildConfig)
	if err != nil {
		return err
	}

	err = ConfigureMessage(buildConfig)
	if err != nil {
		return err
	}

	err = cleanUpBuildDir(buildConfig)
	return err
}

//go:embed scripts/script_base.sh
var scriptBase string

func prepareBuildDir(buildConfig *config.BuildConfig) error {

	/* Combustion works by creating a volume with a subdirectory named "combustion"
	   and a file named "script". This function builds out that structure and updates
	   the BuildConfig so that the other functions can populate it as necessary.
	*/

	if buildConfig.BuildTempDir == "" {
		tmpDir, err := os.MkdirTemp("", "eib-")
		if err != nil {
			return err
		}
		buildConfig.BuildTempDir = tmpDir
	}

	buildConfig.CombustionDir = filepath.Join(buildConfig.BuildTempDir, "combustion")

	err := os.MkdirAll(buildConfig.CombustionDir, os.ModePerm)

	return err
}

func cleanUpBuildDir(buildConfig *config.BuildConfig) error {
	if buildConfig.DeleteArtifacts {
		err := os.Remove(buildConfig.BuildTempDir)
		return err
	}
	return nil
}

func generateCombustionScript(buildConfig *config.BuildConfig) error {

	// The file must be located at "combustion/script"
	scriptFilename := filepath.Join(buildConfig.CombustionDir, "script")
	scriptFile, err := os.Create(scriptFilename)
	if err != nil {
		return err
	}
	defer scriptFile.Close()

	// Write the script initialization lines
	_, err = fmt.Fprintln(scriptFile, scriptBase)
	if err != nil {
		return err
	}

	// Add a call to each script that was added to the combustion directory
	for _, filename := range buildConfig.CombustionScripts {
		_, err = fmt.Fprintln(scriptFile, filename)
		if err != nil {
			return err
		}
	}

	return nil
}
