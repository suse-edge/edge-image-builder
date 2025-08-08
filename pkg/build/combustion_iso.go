package build

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	combustionTmpDir        = "combustion-tmp"
	combustionScriptName    = "create-combustion.sh"
	combustionScriptLogFile = "combustion-build.log"
)

//go:embed templates/generate-combustion-iso.sh.tpl
var combustionScriptTemplate string

func (g *Generator) GenerateCombustionISO() error {
	if err := g.deleteExistingOutputFile(); err != nil {
		return fmt.Errorf("deleting existing combustion image: %w", err)
	}

	if err := g.writeCombustionScript(combustionScriptTemplate, combustionScriptName); err != nil {
		return fmt.Errorf("creating the Combustion extraction script: %w", err)
	}

	return nil
}

func (g *Generator) writeCombustionScript(templateContents, outputFilename string) error {
	scriptName := filepath.Join(g.context.BuildDir, outputFilename)
	combustionTmpPath := filepath.Join(g.context.BuildDir, combustionTmpDir)
	arguments := struct {
		OutputImageFilename string
		CombustionDir       string
		ArtefactsDir        string
		CombustionTmpPath   string
	}{
		OutputImageFilename: g.generateOutputFilename(),
		CombustionDir:       g.context.CombustionDir,
		ArtefactsDir:        g.context.ArtefactsDir,
		CombustionTmpPath:   combustionTmpPath,
	}

	contents, err := template.Parse("combustion-script", templateContents, arguments)
	if err != nil {
		return fmt.Errorf("creating the combustion script from template: %w", err)
	}

	if err = os.WriteFile(scriptName, []byte(contents), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing combustion script %s: %w", outputFilename, err)
	}

	cmd, combustionLog, err := g.generateCombustionCommand(combustionScriptLogFile, combustionScriptName)
	if err != nil {
		return fmt.Errorf("preparing to build the new combustion script: %w", err)
	}
	defer func() {
		if err = combustionLog.Close(); err != nil {
			zap.S().Warnf("failed to close ISO rebuild log file properly: %s", err)
		}
	}()

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("building the new combustion: %w", err)
	}

	return nil
}

// Refactor this into a generic function
func (g *Generator) generateCombustionCommand(logFilename, scriptName string) (*exec.Cmd, *os.File, error) {
	fullLogFilename := filepath.Join(g.context.BuildDir, logFilename)
	logFile, err := os.Create(fullLogFilename)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening combustion log file %s: %w", logFilename, err)
	}

	scriptFilename := filepath.Join(g.context.BuildDir, scriptName)
	cmd := exec.Command(scriptFilename)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	return cmd, logFile, nil
}
