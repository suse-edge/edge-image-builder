package validation

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/combustion"
	"go.uber.org/zap"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	k8sComponent = "Kubernetes"
	httpScheme   = "http"
	httpsScheme  = "https"
	ociScheme    = "oci"
)

var validNodeTypes = []string{image.KubernetesNodeTypeServer, image.KubernetesNodeTypeAgent}

func validateKubernetes(ctx *image.Context) []FailedValidation {
	def := ctx.ImageDefinition

	var failures []FailedValidation

	if !isKubernetesDefined(&def.Kubernetes) {
		return failures
	}

	failures = append(failures, validateNodes(&def.Kubernetes)...)
	failures = append(failures, validateManifestURLs(&def.Kubernetes)...)
	failures = append(failures, validateHelm(&def.Kubernetes, ctx.ImageConfigDir)...)

	return failures
}

func isKubernetesDefined(k8s *image.Kubernetes) bool {
	return k8s.Version != ""
}

func validateNodes(k8s *image.Kubernetes) []FailedValidation {
	var failures []FailedValidation

	numNodes := len(k8s.Nodes)
	if numNodes <= 1 {
		// Single node cluster, node configurations are not required
		return failures
	}

	if k8s.Network.APIVIP == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'apiVIP' field is required in the 'network' section when defining entries under 'nodes'.",
		})
	}

	var nodeTypes []string
	var nodeNames []string
	var initialisers []*image.Node

	for _, node := range k8s.Nodes {
		if node.Hostname == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'hostname' field is required for entries in the 'nodes' section.",
			})
		}

		if node.Type != image.KubernetesNodeTypeServer && node.Type != image.KubernetesNodeTypeAgent {
			options := strings.Join(validNodeTypes, ", ")
			msg := fmt.Sprintf("The 'type' field for entries in the 'nodes' section must be one of: %s", options)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}

		if node.Initialiser {
			n := node
			initialisers = append(initialisers, &n)

			if node.Type == image.KubernetesNodeTypeAgent {
				msg := fmt.Sprintf("The node labeled with 'initialiser' must be of type '%s'.", image.KubernetesNodeTypeServer)
				failures = append(failures, FailedValidation{
					UserMessage: msg,
				})
			}
		}

		nodeNames = append(nodeNames, node.Hostname)
		nodeTypes = append(nodeTypes, node.Type)
	}

	if duplicates := findDuplicates(nodeNames); len(duplicates) > 0 {
		duplicateValues := strings.Join(duplicates, ", ")
		msg := fmt.Sprintf("The 'nodes' section contains duplicate entries: %s", duplicateValues)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	if !slices.Contains(nodeTypes, image.KubernetesNodeTypeServer) {
		msg := fmt.Sprintf("There must be at least one node of type '%s' defined.", image.KubernetesNodeTypeServer)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	if len(initialisers) > 1 {
		failures = append(failures, FailedValidation{
			UserMessage: "Only one node may be specified as the cluster initializer.",
		})
	}

	return failures
}

func validateManifestURLs(k8s *image.Kubernetes) []FailedValidation {
	var failures []FailedValidation

	if len(k8s.Manifests.URLs) == 0 {
		return failures
	}

	seenManifests := make(map[string]bool)
	for _, manifest := range k8s.Manifests.URLs {
		if !strings.HasPrefix(manifest, "http") {
			failures = append(failures, FailedValidation{
				UserMessage: "Entries in 'urls' must begin with either 'http://' or 'https://'.",
			})
		}

		if _, exists := seenManifests[manifest]; exists {
			msg := fmt.Sprintf("The 'urls' field contains duplicate entries: %s", manifest)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}

		seenManifests[manifest] = true
	}

	return failures
}

func validateHelm(k8s *image.Kubernetes, imageConfigDir string) []FailedValidation {
	var failures []FailedValidation

	if len(k8s.Helm.Charts) == 0 {
		return failures
	}

	if len(k8s.Helm.Repositories) == 0 {
		failures = append(failures, FailedValidation{
			UserMessage: "Helm charts defined with no Helm repositories defined.",
		})

		return failures
	}

	var helmRepositoryNames []string
	for _, repo := range k8s.Helm.Repositories {
		helmRepositoryNames = append(helmRepositoryNames, repo.Name)
	}

	if failure := validateHelmChartDuplicates(k8s.Helm.Charts); failure != "" {
		failures = append(failures, FailedValidation{
			UserMessage: failure,
		})
	}

	seenHelmRepos := make(map[string]bool)
	for _, chart := range k8s.Helm.Charts {
		c := chart
		failures = append(failures, validateChart(&c, helmRepositoryNames, imageConfigDir)...)

		seenHelmRepos[chart.RepositoryName] = true
	}

	for _, repo := range k8s.Helm.Repositories {
		r := repo
		failures = append(failures, validateRepo(&r, seenHelmRepos, imageConfigDir)...)
	}

	return failures
}

func validateChart(chart *image.HelmChart, repositoryNames []string, imageConfigDir string) []FailedValidation {
	var failures []FailedValidation

	if chart.Name == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "Helm chart 'name' field must be defined.",
		})
	}

	if chart.RepositoryName == "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm chart 'repositoryName' field for %q must be defined.", chart.Name),
		})
	} else if !slices.Contains(repositoryNames, chart.RepositoryName) {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm chart 'repositoryName' %q for Helm chart %q does not match the name of any defined repository.", chart.RepositoryName, chart.Name),
		})
	}

	if chart.Version == "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm chart 'version' field for %q field must be defined.", chart.Name),
		})
	}

	if chart.CreateNamespace && chart.TargetNamespace == "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm chart 'createNamespace' field for %q cannot be true without 'targetNamespace' being defined.", chart.Name),
		})
	}

	if failure := validateHelmChartValues(chart.Name, chart.ValuesFile, imageConfigDir); failure != "" {
		failures = append(failures, FailedValidation{
			UserMessage: failure,
		})
	}

	return failures
}

func validateRepo(repo *image.HelmRepository, seenHelmRepos map[string]bool, imageConfigDir string) []FailedValidation {
	var failures []FailedValidation

	parsedURL, err := url.Parse(repo.URL)
	if err != nil {
		zap.S().Errorf("Helm repository URL '%s' could not be parsed: %s", repo.URL, err)
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository URL '%s' could not be parsed.", repo.URL),
		})

		return failures
	}

	failures = append(failures, validateHelmRepoName(repo, seenHelmRepos)...)
	failures = append(failures, validateHelmRepoURL(parsedURL, repo)...)
	failures = append(failures, validateHelmRepoAuth(repo)...)
	failures = append(failures, validateHelmRepoArgs(parsedURL, repo)...)

	if failure := validateHelmRepoCert(repo.Name, repo.CAFile, imageConfigDir); failure != "" {
		failures = append(failures, FailedValidation{
			UserMessage: failure,
		})
	}

	return failures
}

func validateHelmRepoName(repo *image.HelmRepository, seenHelmRepos map[string]bool) []FailedValidation {
	var failures []FailedValidation

	if repo.Name == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "Helm repository 'name' field must be defined.",
		})
	} else if !seenHelmRepos[repo.Name] {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'name' field for %q must match the 'repositoryName' field in at least one defined Helm chart.", repo.Name),
		})
	}

	return failures
}

func validateHelmRepoURL(parsedURL *url.URL, repo *image.HelmRepository) []FailedValidation {
	var failures []FailedValidation

	if repo.URL == "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'url' field for %q must be defined.", repo.Name),
		})
	} else if parsedURL.Scheme != httpScheme && parsedURL.Scheme != httpsScheme && parsedURL.Scheme != ociScheme {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'url' field for %q must begin with either 'oci://', 'http://', or 'https://'.", repo.Name),
		})
	}

	return failures
}

func validateHelmRepoAuth(repo *image.HelmRepository) []FailedValidation {
	var failures []FailedValidation

	if repo.Authentication.Username != "" && repo.Authentication.Password == "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'password' field not defined for %q.", repo.Name),
		})
	}

	if repo.Authentication.Username == "" && repo.Authentication.Password != "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'username' field not defined for %q.", repo.Name),
		})
	}

	return failures
}

func validateHelmRepoArgs(parsedURL *url.URL, repo *image.HelmRepository) []FailedValidation {
	var failures []FailedValidation

	if repo.SkipTLSVerify && repo.PlainHTTP {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'plainHTTP' and 'skipTLSVerify' fields for %q cannot both be true.", repo.Name),
		})
	}

	if parsedURL.Scheme == httpScheme && !repo.PlainHTTP {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'url' field for %q contains 'http://' but 'plainHTTP' field is false.", repo.Name),
		})
	}

	if parsedURL.Scheme == httpsScheme && repo.PlainHTTP {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'url' field for %q contains 'https://' but 'plainHTTP' field is true.", repo.Name),
		})
	}

	if parsedURL.Scheme == httpScheme && repo.SkipTLSVerify {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'url' field for %q contains 'http://' but 'skipTLSVerify' field is true.", repo.Name),
		})
	}

	if repo.SkipTLSVerify && repo.CAFile != "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'caFile' field for %q cannot be defined while 'skipTLSVerify' is true.", repo.Name),
		})
	}

	if repo.PlainHTTP && repo.CAFile != "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'caFile' field for %q cannot be defined while 'plainHTTP' is true.", repo.Name),
		})
	}

	if parsedURL.Scheme == httpScheme && repo.CAFile != "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository 'url' field for %q contains 'http://' but 'caFile' field is defined.", repo.Name),
		})
	}

	return failures
}

func validateHelmRepoCert(repoName, certFile string, imageConfigDir string) string {
	if certFile == "" {
		return ""
	}

	validExtensions := []string{".pem", ".crt", ".cer"}
	if !slices.Contains(validExtensions, filepath.Ext(certFile)) {
		return fmt.Sprintf("Helm chart 'caFile' field for %q must be the name of a valid cert file/bundle with one of the following extensions: %s",
			repoName, strings.Join(validExtensions, ", "))
	}

	certFilePath := filepath.Join(imageConfigDir, combustion.K8sDir, combustion.HelmDir, combustion.CertsDir, certFile)
	_, err := os.Stat(certFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Sprintf("Helm repo cert file/bundle '%s' could not be found at '%s'.", certFile, certFilePath)
		}

		zap.S().Errorf("Helm repo cert file/bundle '%s' could not be read: %s", certFile, err)
		return fmt.Sprintf("Helm repo cert file/bundle '%s' could not be read.", certFile)
	}

	return ""
}

func validateHelmChartValues(chartName, valuesFile string, imageConfigDir string) string {
	if valuesFile == "" {
		return ""
	}

	if filepath.Ext(valuesFile) != ".yaml" && filepath.Ext(valuesFile) != ".yml" {
		return fmt.Sprintf("Helm chart 'valuesFile' field for %q must be the name of a valid yaml file ending in '.yaml' or '.yml'.", chartName)
	}

	valuesFilePath := filepath.Join(imageConfigDir, combustion.K8sDir, combustion.HelmDir, combustion.ValuesDir, valuesFile)
	_, err := os.Stat(valuesFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Sprintf("Helm chart values file '%s' could not be found at '%s'.", valuesFile, valuesFilePath)
		}

		zap.S().Errorf("Helm chart values file '%s' could not be read: %s", valuesFile, err)
		return fmt.Sprintf("Helm chart values file '%s' could not be read.", valuesFile)
	}

	return ""
}

func validateHelmChartDuplicates(charts []image.HelmChart) string {
	seenHelmCharts := make(map[string]bool)

	for _, chart := range charts {
		if _, exists := seenHelmCharts[chart.Name]; exists {
			return fmt.Sprintf("The 'helmCharts' field contains duplicate entries: %s", chart.Name)
		}

		seenHelmCharts[chart.Name] = true
	}

	return ""
}
