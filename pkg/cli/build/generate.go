package build

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/cli/cmd"
	"github.com/suse-edge/edge-image-builder/pkg/eib"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func Generate(c *cli.Context) error {
	args := &cmd.CommonArgs

	outputType := c.String("output-type")
	output := c.String("output")
	arch := c.String("arch")

	rootBuildDir := args.RootBuildDir
	if rootBuildDir == "" {
		const defaultBuildDir = "_build"

		rootBuildDir = filepath.Join(args.ConfigDir, defaultBuildDir)
		if err := os.MkdirAll(rootBuildDir, os.ModePerm); err != nil {
			log.Auditf("The root build directory could not be set up under the configuration directory '%s'.", args.ConfigDir)
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

	if cmdErr := imageConfigDirExists(args.ConfigDir); cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	configDriveDefinition, cmdErr := parseDefinitionFile(args.ConfigDir, args.DefinitionFile)
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

	ctx := buildContext(buildDir, combustionDir, artefactsDir, args.ConfigDir, configDriveDefinition, artifactSources)
	ctx.IsConfigDrive = true

	if cmdErr = validateImageDefinition(ctx); cmdErr != nil {
		cmd.LogError(cmdErr, checkBuildLogMessage)
		os.Exit(1)
	}

	// Set the necessary flags for combustion
	ctx.ImageDefinition.Image.ImageType = outputType
	ctx.ImageDefinition.Image.OutputImageName = output

	if strings.EqualFold(arch, string(image.ArchTypeARM)) {
		ctx.ImageDefinition.Image.Arch = image.ArchTypeARM
	} else if strings.EqualFold(arch, string(image.ArchTypeX86)) {
		ctx.ImageDefinition.Image.Arch = image.ArchTypeX86
	}

	defer func() {
		if r := recover(); r != nil {
			log.Auditf("Build failed unexpectedly. %s", checkBuildLogMessage)
			zap.S().Fatalf("Unexpected error occurred: %s", r)
		}
	}()

	if err = eib.Run(ctx, rootBuildDir); err != nil {
		log.Audit(checkBuildLogMessage)
		zap.S().Fatalf("An error occurred building the image: %s", err)
	}

	return nil
}
