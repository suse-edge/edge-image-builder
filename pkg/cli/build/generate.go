package build

import (
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/cli/cmd"
	"github.com/suse-edge/edge-image-builder/pkg/eib"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func Generate(_ *cli.Context) error {
	generateArgs := &cmd.GenerateArgs

	rootBuildDir := generateArgs.GenerateRootBuildDir
	if rootBuildDir == "" {
		const defaultBuildDir = "_build"

		rootBuildDir = filepath.Join(generateArgs.GenerateConfigDir, defaultBuildDir)
		if err := os.MkdirAll(rootBuildDir, os.ModePerm); err != nil {
			log.Auditf("The root build directory could not be set up under the configuration directory '%s'.", generateArgs.GenerateConfigDir)
			return err
		}
	}

	buildDir, err := eib.SetupBuildDirectory(rootBuildDir)
	if err != nil {
		log.Audit("The build directory could not be set up.")
		return err
	}

	// This needs to occur as early as possible so that the subsequent calls can use the log
	log.ConfigureGlobalLogger(filepath.Join(buildDir, buildLogFilename))

	if cmdErr := imageConfigDirExists(generateArgs.GenerateConfigDir); cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	imageDefinition, cmdErr := parseImageDefinition(generateArgs.GenerateConfigDir, generateArgs.GenerateDefinitionFile)
	if cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	combustionDir, artefactsDir, err := eib.SetupCombustionDirectory(buildDir)
	if err != nil {
		log.Auditf("Setting up the combustion directory failed. %s", checkBuildLogMessage)
		zap.S().Fatalf("Failed to create combustion directories: %s", err)
	}

	artifactSources, err := parseArtifactSources()
	if err != nil {
		log.Auditf("Loading artifact sources metadata failed. %s", checkBuildLogMessage)
		zap.S().Fatalf("Parsing artifact sources failed: %v", err)
	}

	ctx := buildContext(buildDir, combustionDir, artefactsDir, generateArgs.GenerateConfigDir, imageDefinition, artifactSources)

	// Set the image type for combustion - either tar or combustion-iso
	ctx.ImageDefinition.Image.ImageType = generateArgs.GenerateOutputType

	if cmdErr = validateImageDefinition(ctx); cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Auditf("Build failed unexpectedly. %s", checkBuildLogMessage)
			zap.S().Fatalf("Unexpected error occurred: %s", r)
		}
	}()

	if err = eib.Generate(ctx, rootBuildDir); err != nil {
		log.Audit(checkBuildLogMessage)
		zap.S().Fatalf("An error occurred building the image: %s", err)
	}

	return nil
}
