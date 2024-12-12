package validation

import (
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/suse-edge/edge-image-builder/pkg/combustion"
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
		failures = append(failures, validateAdditionalArtifacts(ctx)...)
		return failures
	}

	failures = append(failures, validateDualStackConfig(&def.Kubernetes, combustion.KubernetesConfigPath(ctx))...)
	failures = append(failures, validateNetwork(&def.Kubernetes)...)
	failures = append(failures, validateNodes(&def.Kubernetes)...)
	failures = append(failures, validateManifestURLs(&def.Kubernetes)...)
	failures = append(failures, validateHelm(&def.Kubernetes, combustion.HelmValuesPath(ctx), combustion.HelmCertsPath(ctx))...)

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

func validateNetwork(k8s *image.Kubernetes) []FailedValidation {
	var failures []FailedValidation

	if k8s.Network.APIVIP4 == "" && k8s.Network.APIVIP6 == "" {
		if len(k8s.Nodes) > 1 {
			failures = append(failures, FailedValidation{
				UserMessage: "At least one of the (`apiVIP`, `apiVIP6`) fields is required in the 'network' section for multi node clusters.",
			})
		}

		return failures
	}

	if k8s.Network.APIVIP4 != "" {
		ip4, err := netip.ParseAddr(k8s.Network.APIVIP4)
		if err != nil {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Invalid address value %q for field 'apiVIP'.", k8s.Network.APIVIP4),
				Error:       err,
			})

			return failures
		}

		if !ip4.Is4() {
			failures = append(failures, FailedValidation{
				UserMessage: "Only IPv4 addresses are valid for field 'apiVIP'.",
			})
		}

		if !ip4.IsGlobalUnicast() {
			msg := fmt.Sprintf("Non-unicast cluster API address (%s) for field 'apiVIP' is invalid.", k8s.Network.APIVIP4)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}
	}

	if k8s.Network.APIVIP6 != "" {
		ip6, err := netip.ParseAddr(k8s.Network.APIVIP6)
		if err != nil {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Invalid address value %q for field 'apiVIP6'.", k8s.Network.APIVIP6),
				Error:       err,
			})

			return failures
		}

		if !ip6.Is6() {
			failures = append(failures, FailedValidation{
				UserMessage: "Only a IPv6 addresses are valid for field 'apiVIP6'.",
			})
		}

		if !ip6.IsGlobalUnicast() {
			msg := fmt.Sprintf("Non-unicast cluster API address (%s) for field 'apiVIP6' is invalid.", k8s.Network.APIVIP6)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}
	}

	return failures
}

func validateDualStackConfig(k8s *image.Kubernetes, kubernetesConfigPath string) []FailedValidation {
	var failures []FailedValidation

	configFile, err := os.ReadFile(kubernetesConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && (k8s.Network.APIVIP4 != "" && k8s.Network.APIVIP6 != "") {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Kubernetes server config could not be found at '%s,' dual-stack configuration requires a valid cluster-cidr and service-cidr.", kubernetesConfigPath),
			})

			return failures
		} else if k8s.Network.APIVIP4 != "" && k8s.Network.APIVIP6 != "" {
			failures = append(failures, FailedValidation{
				UserMessage: "Kubernetes server config could not be read",
				Error:       err,
			})
		}

		return failures
	}

	serverConfig := map[string]any{}
	if err = yaml.Unmarshal(configFile, &serverConfig); err != nil {
		failures = append(failures, FailedValidation{
			UserMessage: "Parsing kubernetes server config file",
			Error:       err,
		})

		return failures
	}

	failures = append(failures, validateNodeIP(k8s, serverConfig)...)
	failures = append(failures, validateCIDRConfig(k8s, serverConfig)...)

	return failures
}

func validateCIDRConfig(k8s *image.Kubernetes, serverConfig map[string]any) []FailedValidation {
	var failures []FailedValidation

	parsedClusterCIDRs, cidrFailures := parseCIDRs(k8s, serverConfig, "cluster-cidr")
	if len(cidrFailures) != 0 {
		failures = append(failures, cidrFailures...)
		return failures
	}

	parsedServiceCIDRs, cidrFailures := parseCIDRs(k8s, serverConfig, "service-cidr")
	if len(cidrFailures) != 0 {
		failures = append(failures, cidrFailures...)
		return failures
	}

	if len(parsedClusterCIDRs) == 0 && len(parsedServiceCIDRs) == 0 {
		return failures
	}

	if len(parsedServiceCIDRs) != 0 && len(parsedClusterCIDRs) == 0 {
		failures = append(failures, FailedValidation{
			UserMessage: "Kubernetes server config cluster-cidr must be defined when service-cidr is defined",
		})

		return failures
	} else if len(parsedServiceCIDRs) == 0 && len(parsedClusterCIDRs) != 0 {
		failures = append(failures, FailedValidation{
			UserMessage: "Kubernetes server config service-cidr must be defined when cluster-cidr is defined",
		})

		return failures
	}

	clusterCIDRIPv6Priority, clusterCIDRFailures := validateCIDRs(parsedClusterCIDRs, "cluster-cidr")
	if len(clusterCIDRFailures) != 0 {
		failures = append(failures, clusterCIDRFailures...)
	}

	serviceCIDRIPv6Priority, serviceCIDRFailures := validateCIDRs(parsedServiceCIDRs, "service-cidr")
	if len(serviceCIDRFailures) != 0 {
		failures = append(failures, serviceCIDRFailures...)
	}

	if clusterCIDRIPv6Priority && !serviceCIDRIPv6Priority {
		failures = append(failures, FailedValidation{
			UserMessage: "Kubernetes server config cluster-cidr cannot prioritize IPv6 while service-cidr prioritizes IPv4, both must have the same priority",
		})
	} else if !clusterCIDRIPv6Priority && serviceCIDRIPv6Priority {
		failures = append(failures, FailedValidation{
			UserMessage: "Kubernetes server config cluster-cidr cannot prioritize IPv4 while service-cidr prioritizes IPv6, both must have the same priority",
		})
	}

	return failures
}

func parseCIDRs(k8s *image.Kubernetes, serverConfig map[string]any, cidrField string) ([]string, []FailedValidation) {
	var failures []FailedValidation
	var parsedCIDRs []string
	if cidr, ok := serverConfig[cidrField].(string); ok {
		parsedCIDRs = strings.Split(cidr, ",")
		if len(parsedCIDRs) != 2 && k8s.Network.APIVIP4 != "" && k8s.Network.APIVIP6 != "" {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Kubernetes server config %s required for a dual-stack cluster", cidrField),
			})
			return nil, failures
		}
	} else if k8s.Network.APIVIP4 != "" && k8s.Network.APIVIP6 != "" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Kubernetes server config must contain a valid %s when configuring dual-stack", cidrField),
		})
		return nil, failures
	}

	return parsedCIDRs, failures
}

func validateCIDRs(parsedCIDRs []string, cidrType string) (bool, []FailedValidation) {
	var failures []FailedValidation

	firstCIDR := parsedCIDRs[0]

	firstCIDRIP, ipFailures := validateIP(firstCIDR, cidrType, true)
	if len(ipFailures) != 0 {
		failures = append(failures, ipFailures...)
		return false, failures
	}

	var secondCIDRIP *netip.Addr
	if len(parsedCIDRs) > 1 {
		secondCIDR := parsedCIDRs[1]
		secondCIDRIP, ipFailures = validateIP(secondCIDR, cidrType, true)
		if len(ipFailures) != 0 {
			failures = append(failures, ipFailures...)
			return false, failures
		}

		if firstCIDRIP.Is4() && secondCIDRIP.Is4() {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Kubernetes server config %s values '%s' and %s cannot both be IPv4, one must be IPv6", cidrType, firstCIDR, secondCIDR),
			})

			return false, failures
		}

		if firstCIDRIP.Is6() && secondCIDRIP.Is6() {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Kubernetes server config %s values '%s' and %s cannot both be IPv6, one must be IPv4", cidrType, firstCIDR, secondCIDR),
			})
			return false, failures
		}
	}

	return firstCIDRIP.Is6(), failures
}

func validateNodeIP(k8s *image.Kubernetes, serverConfig map[string]any) []FailedValidation {
	var failures []FailedValidation

	if nodeIPs, ok := serverConfig["node-ip"].(string); ok {
		if len(k8s.Nodes) > 1 {
			var serverCount int
			for _, nodes := range k8s.Nodes {
				if nodes.Type == "server" {
					serverCount++
				}
			}

			if serverCount > 1 {
				failures = append(failures, FailedValidation{
					UserMessage: "Kubernetes server config node-ip can not be specified when there is more than one Kubernetes server node",
				})
				return failures
			}
		}

		parsedNodeIPs := strings.Split(nodeIPs, ",")
		switch len(parsedNodeIPs) {
		case 1:
			_, ipFailures := validateIP(parsedNodeIPs[0], "node-ip", false)
			if len(ipFailures) != 0 {
				failures = append(failures, ipFailures...)
				return failures
			}
		case 2:
			ip1, ipFailures := validateIP(parsedNodeIPs[0], "node-ip", false)
			if len(ipFailures) != 0 {
				failures = append(failures, ipFailures...)
				return failures
			}

			ip2, ipFailures := validateIP(parsedNodeIPs[1], "node-ip", false)
			if len(ipFailures) != 0 {
				failures = append(failures, ipFailures...)
				return failures
			}

			if (ip1.Is4() && ip2.Is4()) || (ip1.Is6() && ip2.Is6()) {
				failures = append(failures, FailedValidation{
					UserMessage: fmt.Sprintf("Kubernetes server config node-ip %s and %s cannot both be of the same IP address family, one must be IPv4, and the other IPv6", parsedNodeIPs[0], parsedNodeIPs[1]),
				})
				return failures
			}
		default:
			failures = append(failures, FailedValidation{
				UserMessage: "Kubernetes server config node-ip cannot contain more than one IPv4 IP and/or one IPv6 IP",
			})
			return failures

		}
	}

	return failures
}

func validateIP(ip string, configField string, hasPrefix bool) (*netip.Addr, []FailedValidation) {
	var failures []FailedValidation
	var extractedIP netip.Addr

	if hasPrefix {
		parsedPrefix, err := netip.ParsePrefix(ip)
		if err != nil {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Kubernetes server config %s value '%s' could not be parsed", configField, ip),
				Error:       err,
			})
			return nil, failures
		}

		extractedIP = parsedPrefix.Addr()
	} else {
		parsedIP, err := netip.ParseAddr(ip)
		if err != nil {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Kubernetes server config %s value '%s' could not be parsed", configField, ip),
				Error:       err,
			})
			return nil, failures
		}
		extractedIP = parsedIP
	}

	if !extractedIP.IsGlobalUnicast() && !hasPrefix {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Kubernetes server config node-ip %s must be a valid unicast IP", ip),
		})
		return nil, failures
	} else if !extractedIP.IsGlobalUnicast() && hasPrefix {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Kubernetes server config %s value '%s' must be a valid unicast address with a prefix", configField, ip),
		})
		return nil, failures
	}

	return &extractedIP, failures
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

func validateHelm(k8s *image.Kubernetes, valuesDir, certsDir string) []FailedValidation {
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
	for i := range k8s.Helm.Charts {
		failures = append(failures, validateChart(&k8s.Helm.Charts[i], helmRepositoryNames, valuesDir)...)

		seenHelmRepos[k8s.Helm.Charts[i].RepositoryName] = true
	}

	for _, repo := range k8s.Helm.Repositories {
		r := repo
		failures = append(failures, validateRepo(&r, seenHelmRepos, certsDir)...)
	}

	return failures
}

func validateChart(chart *image.HelmChart, repositoryNames []string, valuesDir string) []FailedValidation {
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

	failures = append(failures, validateHelmChartValues(chart.Name, chart.ValuesFile, valuesDir)...)

	return failures
}

func validateRepo(repo *image.HelmRepository, seenHelmRepos map[string]bool, certsDir string) []FailedValidation {
	var failures []FailedValidation

	parsedURL, err := url.Parse(repo.URL)
	if err != nil {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm repository URL '%s' could not be parsed.", repo.URL),
			Error:       err,
		})

		return failures
	}

	failures = append(failures, validateHelmRepoName(repo, seenHelmRepos)...)
	failures = append(failures, validateHelmRepoURL(parsedURL, repo)...)
	failures = append(failures, validateHelmRepoAuth(repo)...)
	failures = append(failures, validateHelmRepoArgs(parsedURL, repo)...)
	failures = append(failures, validateHelmRepoCert(repo.Name, repo.CAFile, certsDir)...)

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

func validateHelmRepoCert(repoName, certFile, certsDir string) []FailedValidation {
	if certFile == "" {
		return nil
	}

	var failures []FailedValidation

	validExtensions := []string{".pem", ".crt", ".cer"}
	if !slices.Contains(validExtensions, filepath.Ext(certFile)) {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm chart 'caFile' field for %q must be the name of a valid cert file/bundle with one of the following extensions: %s",
				repoName, strings.Join(validExtensions, ", ")),
		})
		return failures
	}

	certFilePath := filepath.Join(certsDir, certFile)
	_, err := os.Stat(certFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Helm repo cert file/bundle '%s' could not be found at '%s'.", certFile, certFilePath),
			})
		} else {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Helm repo cert file/bundle '%s' could not be read", certFile),
				Error:       err,
			})
		}
	}

	return failures
}

func validateHelmChartValues(chartName, valuesFile, valuesDir string) []FailedValidation {
	if valuesFile == "" {
		return nil
	}

	var failures []FailedValidation

	if filepath.Ext(valuesFile) != ".yaml" && filepath.Ext(valuesFile) != ".yml" {
		failures = append(failures, FailedValidation{
			UserMessage: fmt.Sprintf("Helm chart 'valuesFile' field for %q must be the name of a valid yaml file ending in '.yaml' or '.yml'.", chartName),
		})
		return failures
	}

	valuesFilePath := filepath.Join(valuesDir, valuesFile)
	_, err := os.Stat(valuesFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Helm chart values file '%s' could not be found at '%s'.", valuesFile, valuesFilePath),
			})
		} else {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Helm chart values file '%s' could not be read.", valuesFile),
				Error:       err,
			})
		}
	}

	return failures
}

func validateHelmChartDuplicates(charts []image.HelmChart) string {
	seenHelmCharts := make(map[string]bool)

	for i := range charts {
		if _, exists := seenHelmCharts[charts[i].Name]; exists {
			return fmt.Sprintf("The 'helmCharts' field contains duplicate entries: %s", charts[i].Name)
		}

		seenHelmCharts[charts[i].Name] = true
	}

	return ""
}

func validateAdditionalArtifacts(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation

	dirEntries, err := os.ReadDir(combustion.KubernetesManifestsPath(ctx))
	if err != nil && !os.IsNotExist(err) {
		failures = append(failures, FailedValidation{
			UserMessage: "Kubernetes manifests directory could not be read",
			Error:       err,
		})
	}

	if len(dirEntries) != 0 {
		failures = append(failures, FailedValidation{
			UserMessage: "Kubernetes version must be defined when local manifests are configured",
		})
	}

	if len(ctx.ImageDefinition.Kubernetes.Helm.Charts) != 0 {
		failures = append(failures, FailedValidation{
			UserMessage: "Kubernetes version must be defined when Helm charts are specified",
		})
	}
	if len(ctx.ImageDefinition.Kubernetes.Manifests.URLs) != 0 {
		failures = append(failures, FailedValidation{
			UserMessage: "Kubernetes version must be defined when manifest URLs are specified",
		})
	}

	return failures
}
