package helm

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"

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
	outputDir string
}

func New(outputDir string) *Helm {
	return &Helm{
		outputDir: outputDir,
	}
}

func repositoryName(repoName, repoURL, chart string) string {
	if strings.HasPrefix(repoURL, "http") {
		return fmt.Sprintf("%s/%s", repoName, chart)
	}

	return repoURL
}

func (h *Helm) AddRepo(repo *image.HelmRepository) error {
	if !strings.HasPrefix(repo.URL, "http") {
		zap.S().Infof("Skipping 'helm repo add' for non-http(s) repository: %s", repo.Name)
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

	cmd := addRepoCommand(repo, file)

	if _, err = fmt.Fprintf(file, "command: %s\n", cmd); err != nil {
		return fmt.Errorf("writing command prefix to log file: %w", err)
	}

	return cmd.Run()
}

func addRepoCommand(repo *image.HelmRepository, output io.Writer) *exec.Cmd {
	var args []string
	args = append(args, "repo", "add", repo.Name, repo.URL)

	cmd := exec.Command("helm", args...)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd
}

func (h *Helm) Pull(chart string, repo *image.HelmRepository, version, destDir string) (string, error) {
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

	cmd := pullCommand(chart, repo, version, destDir, file)

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

func pullCommand(chart string, repo *image.HelmRepository, version, destDir string, output io.Writer) *exec.Cmd {
	repository := repositoryName(repo.Name, repo.URL, chart)

	var args []string
	args = append(args, "pull", repository)

	if version != "" {
		args = append(args, "--version", version)
	}
	if destDir != "" {
		args = append(args, "--destination", destDir)
	}

	cmd := exec.Command("helm", args...)

	cmd.Stdout = output
	cmd.Stderr = output

	return cmd
}

func (h *Helm) Template(chart, repository, version, valuesFilePath, kubeVersion string) ([]map[string]any, error) {
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
	cmd := templateCommand(chart, repository, version, valuesFilePath, kubeVersion, io.MultiWriter(file, chartContentsBuffer), file)

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

func templateCommand(chart, repository, version, valuesFilePath, kubeVersion string, stdout, stderr io.Writer) *exec.Cmd {
	var args []string
	args = append(args, "template", "--skip-crds", chart, repository)

	if version != "" {
		args = append(args, "--version", version)
	}

	if valuesFilePath != "" {
		args = append(args, "-f", valuesFilePath)
	}

	args = append(args, "--kube-version", kubeVersion)

	cmd := exec.Command("helm", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd
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
