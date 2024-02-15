package helm

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	templateLogFileName = "helm-template.log"
	pullLogFileName     = "helm-pull.log"
	repoAddLogFileName  = "helm-repo-add.log"

	outputFileFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
)

type Helm struct {
	execCommand func(name string, args ...string) *exec.Cmd
	outputDir   string
}

func New(outputDir string) *Helm {
	return &Helm{
		execCommand: exec.Command,
		outputDir:   outputDir,
	}
}

func tempRepo(chart string) string {
	return fmt.Sprintf("repo-%s", chart)
}

func repositoryName(repoURL, chart string) string {
	if strings.HasPrefix(repoURL, "http") {
		return fmt.Sprintf("%s/%s", tempRepo(chart), chart)
	}

	return repoURL
}

func (h *Helm) AddRepo(chart, repository string) error {
	if !strings.HasPrefix(repository, "http") {
		zap.S().Infof("Skipping 'helm repo add' for non-http(s) repository: %s", repository)
		return nil
	}

	logFile := filepath.Join(h.outputDir, repoAddLogFileName)

	file, err := os.OpenFile(logFile, outputFileFlags, fileio.NonExecutablePerms)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			zap.S().Warnf("Closing %s file failed: %s", logFile, err)
		}
	}()

	var args []string
	args = append(args, "repo", "add", tempRepo(chart), repository)

	cmd := h.execCommand("helm", args...)
	cmd.Stdout = file
	cmd.Stderr = file

	if _, err = fmt.Fprintf(file, "command: %s\n", cmd); err != nil {
		return fmt.Errorf("writing command prefix to log file: %w", err)
	}

	return cmd.Run()
}

func (h *Helm) Pull(chart, repository, version, destDir string) (string, error) {
	logFile := filepath.Join(h.outputDir, pullLogFileName)

	file, err := os.OpenFile(logFile, outputFileFlags, fileio.NonExecutablePerms)
	if err != nil {
		return "", fmt.Errorf("opening log file: %w", err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			zap.S().Warnf("Closing %s file failed: %s", logFile, err)
		}
	}()

	repository = repositoryName(repository, chart)

	var args []string
	args = append(args, "pull", repository)

	if version != "" {
		args = append(args, "--version", version)
	}
	if destDir != "" {
		args = append(args, "--destination", destDir)
	}

	cmd := h.execCommand("helm", args...)
	cmd.Stdout = file
	cmd.Stderr = file

	if _, err = fmt.Fprintf(file, "command: %s\n", cmd); err != nil {
		return "", fmt.Errorf("writing command prefix to log file: %w", err)
	}

	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("executing command: %w", err)
	}

	chartPathPattern := fmt.Sprintf("%s-*.tgz", filepath.Join(destDir, chart))

	matches, err := filepath.Glob(chartPathPattern)
	if err != nil {
		return "", fmt.Errorf("looking for chart with pattern %s: %w", chartPathPattern, err)
	} else if len(matches) != 1 {
		return "", fmt.Errorf("unable to locate downloaded chart: %s", chart)
	}

	chartPath := matches[0]
	return chartPath, nil
}

func (h *Helm) Template(chart, repository, version, valuesFilePath string, setArgs []string) ([]map[string]any, error) {
	logFile := filepath.Join(h.outputDir, templateLogFileName)

	file, err := os.OpenFile(logFile, outputFileFlags, fileio.NonExecutablePerms)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			zap.S().Warnf("Closing %s file failed: %s", logFile, err)
		}
	}()

	chartContentsBuffer := new(strings.Builder)

	var args []string
	args = append(args, "template", "--skip-crds", chart, repository)

	if version != "" {
		args = append(args, "--version", version)
	}

	if len(setArgs) > 0 {
		args = append(args, "--set", strings.Join(setArgs, ","))
	}

	if valuesFilePath != "" {
		args = append(args, "-f", valuesFilePath)
	}

	cmd := h.execCommand("helm", args...)
	cmd.Stdout = io.MultiWriter(file, chartContentsBuffer)
	cmd.Stderr = file

	if _, err = fmt.Fprintf(file, "command: %s\n", cmd); err != nil {
		return nil, fmt.Errorf("writing command prefix to log file: %w", err)
	}

	if err = cmd.Run(); err != nil {
		return nil, fmt.Errorf("executing command: %w", err)
	}

	chartContents := chartContentsBuffer.String()
	resources, err := parseChartContents(chartContents)
	if err != nil {
		return nil, fmt.Errorf("parsing chart contents: %w", err)
	}

	return resources, nil
}

func parseChartContents(chartContents string) ([]map[string]any, error) {
	var resources []map[string]any

	for _, resource := range strings.Split(chartContents, "---") {
		if resource == "" {
			continue
		}

		source, content, found := strings.Cut(resource, "\n")
		if !found {
			return nil, fmt.Errorf("invalid resource: %s", resource)
		}

		var r map[string]any
		if err := yaml.Unmarshal([]byte(content), &r); err != nil {
			return nil, fmt.Errorf("decoding resource from source '%s': %w", source, err)
		}

		resources = append(resources, r)
	}

	return resources, nil
}
