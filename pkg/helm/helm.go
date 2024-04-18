package helm

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	templateLogFileName   = "helm-template.log"
	pullLogFileName       = "helm-pull.log"
	repoAddLogFileName    = "helm-repo-add.log"
	registryLoginFileName = "helm-registry-login.log"

	outputFileFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
)

type Helm struct {
	outputDir string
	certsDir  string
}

func New(outputDir, certsDir string) *Helm {
	return &Helm{
		outputDir: outputDir,
		certsDir:  certsDir,
	}
}

func chartPath(repoName, repoURL, chart string) string {
	if strings.HasPrefix(repoURL, "http") {
		return fmt.Sprintf("%s/%s", repoName, chart)
	}

	path, _ := url.JoinPath(repoURL, chart)
	return path
}

func (h *Helm) AddRepo(repo *image.HelmRepository) error {
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

	cmd := addRepoCommand(repo, h.certsDir, file)

	if _, err = fmt.Fprintf(file, "command: %s\n", cmd); err != nil {
		return fmt.Errorf("writing command prefix to log file: %w", err)
	}

	return cmd.Run()
}

func addRepoCommand(repo *image.HelmRepository, certsDir string, output io.Writer) *exec.Cmd {
	var args []string
	args = append(args, "repo", "add", repo.Name, repo.URL)

	if repo.Authentication.Username != "" && repo.Authentication.Password != "" {
		args = append(args, "--username", repo.Authentication.Username, "--password", repo.Authentication.Password)
	}

	if repo.SkipTLSVerify {
		args = append(args, "--insecure-skip-tls-verify")
	} else if repo.CAFile != "" {
		caFilePath := filepath.Join(certsDir, repo.CAFile)
		args = append(args, "--ca-file", caFilePath)
	}

	cmd := exec.Command("helm", args...)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd
}

func (h *Helm) RegistryLogin(repo *image.HelmRepository) error {
	logFile := filepath.Join(h.outputDir, registryLoginFileName)

	file, err := os.OpenFile(logFile, outputFileFlags, fileio.NonExecutablePerms)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			zap.S().Warnf("Closing %s file failed: %s", logFile, err)
		}
	}()

	host, err := getHost(repo.URL)
	if err != nil {
		return fmt.Errorf("getting host url: %w", err)
	}

	cmd := registryLoginCommand(host, repo, h.certsDir, file)

	if _, err = fmt.Fprintf(file, "command: %s\n", cmd); err != nil {
		return fmt.Errorf("writing command prefix to log file: %w", err)
	}

	return cmd.Run()
}

func registryLoginCommand(host string, repo *image.HelmRepository, certsDir string, output io.Writer) *exec.Cmd {
	var args []string
	args = append(args, "registry", "login", host)

	if repo.Authentication.Username != "" && repo.Authentication.Password != "" {
		args = append(args, "--username", repo.Authentication.Username, "--password", repo.Authentication.Password)
	}

	if repo.SkipTLSVerify || repo.PlainHTTP {
		args = append(args, "--insecure")
	} else if repo.CAFile != "" {
		caFilePath := filepath.Join(certsDir, repo.CAFile)
		args = append(args, "--ca-file", caFilePath)
	}

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

	chartDir := filepath.Join(destDir, chart)
	if err = os.MkdirAll(chartDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("creating chart dir %q: %w", chartDir, err)
	}

	cmd := pullCommand(chart, repo, version, chartDir, h.certsDir, file)

	if _, err = fmt.Fprintf(file, "command: %s\n", cmd); err != nil {
		return "", fmt.Errorf("writing command prefix to log file: %w", err)
	}

	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("executing command: %w", err)
	}

	chartPathPattern := fmt.Sprintf("%s/%s-*.tgz", chartDir, chart)

	matches, err := filepath.Glob(chartPathPattern)
	if err != nil {
		return "", fmt.Errorf("looking for chart with pattern %s: %w", chartPathPattern, err)
	} else if len(matches) != 1 {
		return "", fmt.Errorf("unable to locate downloaded chart: %s", chart)
	}

	chartPath := matches[0]
	return chartPath, nil
}

func pullCommand(chart string, repo *image.HelmRepository, version, destDir, certsDir string, output io.Writer) *exec.Cmd {
	path := chartPath(repo.Name, repo.URL, chart)

	var args []string
	args = append(args, "pull", path)

	if version != "" {
		args = append(args, "--version", version)
	}
	if destDir != "" {
		args = append(args, "--destination", destDir)
	}

	switch {
	case repo.SkipTLSVerify:
		args = append(args, "--insecure-skip-tls-verify")
	case repo.PlainHTTP:
		args = append(args, "--plain-http")
	case repo.CAFile != "":
		caFilePath := filepath.Join(certsDir, repo.CAFile)
		args = append(args, "--ca-file", caFilePath)
	}

	cmd := exec.Command("helm", args...)

	cmd.Stdout = output
	cmd.Stderr = output

	return cmd
}

func (h *Helm) Template(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string) ([]map[string]any, error) {
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
	cmd := templateCommand(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace, io.MultiWriter(file, chartContentsBuffer), file)

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

func templateCommand(chart, repository, version, valuesFilePath, kubeVersion, targetNamespace string, stdout, stderr io.Writer) *exec.Cmd {
	var args []string
	args = append(args, "template", "--skip-crds", chart, repository)

	if targetNamespace != "" {
		args = append(args, "--namespace", targetNamespace)
	}

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

	for _, resource := range strings.Split(chartContents, "---\n") {
		if resource == "" {
			continue
		}

		resource = strings.TrimSpace(resource)
		if !strings.HasPrefix(resource, "# Source") {
			continue
		}

		source, content, found := strings.Cut(resource, "\n")
		if !found {
			zap.S().Warnf("Invalid Helm resource: %s", resource)
			continue
		}

		var r map[string]any
		if err := yaml.Unmarshal([]byte(content), &r); err != nil {
			return nil, fmt.Errorf("decoding resource from source '%s': %w", source, err)
		}

		resources = append(resources, r)
	}

	return resources, nil
}

func getHost(repoURL string) (string, error) {
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("parsing url %q: %w", repoURL, err)
	}

	return parsedURL.Host, nil
}
