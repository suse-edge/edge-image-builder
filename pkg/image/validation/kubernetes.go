package validation

import (
	"fmt"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	k8sComponent = "Kubernetes"
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

	if k8s.Network.APIHost == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'apiHost' field is required in the 'network' section when defining entries under 'nodes'.",
		})
	}

	var nodeTypes []string
	var nodeNames []string
	var firstNodes []*image.Node

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

		if node.First {
			n := node
			firstNodes = append(firstNodes, &n)

			if node.Type == image.KubernetesNodeTypeAgent {
				_ = node.First // Jan 19, 2024: The word "first" is being discussed; tf this field gets renamed, update the text below
				msg := fmt.Sprintf("The node labeled with 'firstNode' must be of type '%s'.", image.KubernetesNodeTypeServer)
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

	if len(firstNodes) > 1 {
		_ = firstNodes[0].First // Jan 19, 2024: The word "first" is being discussed; tf this field gets renamed, update the text below
		failures = append(failures, FailedValidation{
			UserMessage: "Only one node may be specified as the cluster initializer (by including the 'firstNode' field).",
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
