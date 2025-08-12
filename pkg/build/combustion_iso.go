package build

import (
	"fmt"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	combustionTmpDir        = "combustion-tmp"
	combustionScriptLogFile = "combustion-build.log"
)

func (g *Generator) generateCombustionISO() error {
	if err := deleteFile(g.context.OutputPath()); err != nil {
		return fmt.Errorf("deleting existing combustion image: %w", err)
	}

	if err := g.createCombustionISO(); err != nil {
		return fmt.Errorf("building combustion ISO: %w", err)
	}

	return nil
}

func (g *Generator) createCombustionISO() error {
	combustionPath := filepath.Join(g.context.BuildDir, combustionTmpDir)
	if err := os.MkdirAll(combustionPath, 0755); err != nil {
		return fmt.Errorf("creating temp directory %s: %w", combustionPath, err)
	}

	combustionDestPath := filepath.Join(combustionPath, filepath.Base(g.context.CombustionDir))
	if err := fileio.CopyFiles(g.context.CombustionDir, combustionDestPath, "", true, nil); err != nil {
		return fmt.Errorf("copying combustion directory: %w", err)
	}

	artefactsDestPath := filepath.Join(combustionPath, filepath.Base(g.context.ArtefactsDir))
	if err := fileio.CopyFiles(g.context.ArtefactsDir, artefactsDestPath, "", true, nil); err != nil {
		return fmt.Errorf("copying artefacts directory: %w", err)
	}

	logFilename := filepath.Join(g.context.BuildDir, combustionScriptLogFile)
	logFile, err := os.Create(logFilename)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	defer func() {
		if err := logFile.Close(); err != nil {
			zap.S().Warnf("Failed to close log file properly: %v", err)
		}
	}()

	if err := g.createISO(combustionPath, logFile); err != nil {
		return fmt.Errorf("creating ISO: %w", err)
	}

	return nil
}

func (g *Generator) createISO(sourcePath string, logFile io.Writer) error {
	outputPath := g.context.OutputPath()

	cmd := exec.Command("mkisofs", "-J", "-o", outputPath, "-V", "COMBUSTION", sourcePath)

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd.Run()
}
