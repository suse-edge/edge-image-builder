package validation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

var validNetwork = image.Network{
	APIHost: "host.com",
	APIVIP:  "127.0.0.1",
}

func TestValidateKubernetes(t *testing.T) {
	tests := map[string]struct {
		K8s                    image.Kubernetes
		ExpectedFailedMessages []string
	}{
		`not defined`: {
			K8s: image.Kubernetes{},
		},
		`all valid`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname:    "server",
						Type:        image.KubernetesNodeTypeServer,
						Initialiser: true,
					},
					{
						Hostname: "agent1",
						Type:     image.KubernetesNodeTypeAgent,
					},
				},
				HelmCharts: []image.HelmChart{
					{
						Name:                  "apache",
						Repo:                  "oci://registry-1.docker.io/bitnamicharts/apache",
						TargetNamespace:       "web",
						CreateNamespace:       true,
						InstallationNamespace: "kube-system",
						Version:               "10.7.0",
						ValuesFile:            "apache-values.yaml",
					},
				},
			},
		},
		`failures all sections`: {
			K8s: image.Kubernetes{
				Version: "1.0",
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Type:        image.KubernetesNodeTypeServer,
						Initialiser: true,
					},
					{
						Hostname: "valid",
						Type:     image.KubernetesNodeTypeAgent,
					},
				},
				Manifests: image.Manifests{
					URLs: []string{
						"example.com",
					},
				},
				HelmCharts: []image.HelmChart{
					{
						Name:    "",
						Repo:    "oci://registry-1.docker.io/bitnamicharts/apache",
						Version: "10.7.0",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'hostname' field is required for entries in the 'nodes' section.",
				"Entries in 'urls' must begin with either 'http://' or 'https://'.",
				"Helm Chart 'name' field must be defined.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := image.Context{
				ImageDefinition: &image.Definition{
					Kubernetes: test.K8s,
				},
			}
			failures := validateKubernetes(&ctx)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}

		})
	}
}

func TestIsKubernetesDefined(t *testing.T) {
	result := isKubernetesDefined(&image.Kubernetes{
		Version: "1.0",
	})
	assert.True(t, result)

	result = isKubernetesDefined(&image.Kubernetes{
		Network:    image.Network{},
		Nodes:      []image.Node{},
		Manifests:  image.Manifests{},
		HelmCharts: []image.HelmChart{},
	})
	assert.False(t, result)
}

func TestValidateNodes(t *testing.T) {
	tests := map[string]struct {
		K8s                    image.Kubernetes
		ExpectedFailedMessages []string
	}{
		`valid`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname: "agent1",
						Type:     image.KubernetesNodeTypeAgent,
					},
					{
						Hostname:    "server",
						Type:        image.KubernetesNodeTypeServer,
						Initialiser: true,
					},
				},
			},
		},
		`no nodes`: {
			K8s: image.Kubernetes{
				Nodes: []image.Node{},
			},
		},
		`with nodes - no network config`: {
			K8s: image.Kubernetes{
				Network: image.Network{},
				Nodes: []image.Node{
					{
						Hostname: "host1",
						Type:     image.KubernetesNodeTypeServer,
					},
					{
						Hostname: "host2",
						Type:     image.KubernetesNodeTypeAgent,
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'apiVIP' field is required in the 'network' section when defining entries under 'nodes'.",
			},
		},
		`no hostname`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname: "host1",
						Type:     image.KubernetesNodeTypeServer,
					},
					{
						Type: image.KubernetesNodeTypeServer,
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'hostname' field is required for entries in the 'nodes' section.",
			},
		},
		`missing type`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname: "host1",
						Type:     image.KubernetesNodeTypeServer,
					},
					{
						Hostname: "valid",
					},
				},
			},
			ExpectedFailedMessages: []string{
				fmt.Sprintf("The 'type' field for entries in the 'nodes' section must be one of: %s", strings.Join(validNodeTypes, ", ")),
			},
		},
		`invalid type`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname: "valid",
						Type:     image.KubernetesNodeTypeServer,
					},
					{
						Hostname: "invalid",
						Type:     "abnormal",
					},
				},
			},
			ExpectedFailedMessages: []string{
				fmt.Sprintf("The 'type' field for entries in the 'nodes' section must be one of: %s", strings.Join(validNodeTypes, ", ")),
			},
		},
		`incorrect initialiser type`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname: "valid",
						Type:     image.KubernetesNodeTypeServer,
					},
					{
						Hostname:    "invalid",
						Initialiser: true,
						Type:        image.KubernetesNodeTypeAgent,
					},
				},
			},
			ExpectedFailedMessages: []string{
				fmt.Sprintf("The node labeled with 'initialiser' must be of type '%s'.", image.KubernetesNodeTypeServer),
			},
		},
		`duplicate entries`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname:    "foo",
						Type:        image.KubernetesNodeTypeServer,
						Initialiser: true,
					},
					{
						Hostname: "bar",
						Type:     image.KubernetesNodeTypeAgent,
					},
					{
						Hostname: "bar",
						Type:     image.KubernetesNodeTypeAgent,
					},
					{
						Hostname: "foo",
						Type:     image.KubernetesNodeTypeAgent,
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'nodes' section contains duplicate entries: bar, foo",
			},
		},
		`no server node`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname: "foo",
						Type:     image.KubernetesNodeTypeAgent,
					},
					{
						Hostname: "bar",
						Type:     image.KubernetesNodeTypeAgent,
					},
				},
			},
			ExpectedFailedMessages: []string{
				fmt.Sprintf("There must be at least one node of type '%s' defined.", image.KubernetesNodeTypeServer),
			},
		},
		`multiple initialisers`: {
			K8s: image.Kubernetes{
				Network: validNetwork,
				Nodes: []image.Node{
					{
						Hostname:    "foo",
						Type:        image.KubernetesNodeTypeServer,
						Initialiser: true,
					},
					{
						Hostname:    "bar",
						Type:        image.KubernetesNodeTypeServer,
						Initialiser: true,
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Only one node may be specified as the cluster initializer.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			k := test.K8s
			failures := validateNodes(&k)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}

		})
	}
}

func TestValidateManifestURLs(t *testing.T) {
	tests := map[string]struct {
		K8s                    image.Kubernetes
		ExpectedFailedMessages []string
	}{
		`valid`: {
			K8s: image.Kubernetes{
				Manifests: image.Manifests{
					URLs: []string{
						"http://valid1.com",
						"https://valid2.com",
					},
				},
			},
		},
		`no URLs`: {
			K8s: image.Kubernetes{
				Manifests: image.Manifests{},
			},
		},
		`invalid prefix`: {
			K8s: image.Kubernetes{
				Manifests: image.Manifests{
					URLs: []string{
						"http://valid.com",
						"https://also-valid.com",
						"invalid.com",
						"nope.com",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Entries in 'urls' must begin with either 'http://' or 'https://'.",
				"Entries in 'urls' must begin with either 'http://' or 'https://'.",
			},
		},
		`duplicate URLs`: {
			K8s: image.Kubernetes{
				Manifests: image.Manifests{
					URLs: []string{
						"http://foo.com",
						"http://bar.com",
						"http://foo.com",
						"http://bar.com",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'urls' field contains duplicate entries: http://foo.com",
				"The 'urls' field contains duplicate entries: http://bar.com",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			k := test.K8s
			failures := validateManifestURLs(&k)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestValidateHelmCharts(t *testing.T) {
	tests := map[string]struct {
		K8s                    image.Kubernetes
		ExpectedFailedMessages []string
	}{
		`valid`: {
			K8s: image.Kubernetes{
				HelmCharts: []image.HelmChart{
					{
						Name:                  "apache",
						Repo:                  "oci://registry-1.docker.io/bitnamicharts/apache",
						TargetNamespace:       "web",
						CreateNamespace:       true,
						InstallationNamespace: "kube-system",
						Version:               "10.7.0",
					},
				},
			},
		},
		`no name`: {
			K8s: image.Kubernetes{
				HelmCharts: []image.HelmChart{
					{
						Name:    "",
						Repo:    "oci://registry-1.docker.io/bitnamicharts/apache",
						Version: "10.7.0",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm Chart 'name' field must be defined.",
			},
		},
		`duplicate name`: {
			K8s: image.Kubernetes{
				HelmCharts: []image.HelmChart{
					{
						Name:    "apache",
						Repo:    "oci://registry-1.docker.io/bitnamicharts/apache",
						Version: "10.7.0",
					},
					{
						Name:    "apache",
						Repo:    "oci://registry-1.docker.io/bitnamicharts/apache",
						Version: "10.7.0",
					},
					{
						Name:    "metallb",
						Repo:    "https://suse-edge.github.io/charts",
						Version: "0.13.10",
					},
					{
						Name:    "metallb",
						Repo:    "https://suse-edge.github.io/charts",
						Version: "0.13.10",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'helmCharts' field contains duplicate entries: apache",
				"The 'helmCharts' field contains duplicate entries: metallb",
			},
		},
		`no repo`: {
			K8s: image.Kubernetes{
				HelmCharts: []image.HelmChart{
					{
						Name:    "apache",
						Version: "10.7.0",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm Chart 'repo' field must be defined.",
			},
		},
		`no version`: {
			K8s: image.Kubernetes{
				HelmCharts: []image.HelmChart{
					{
						Name:    "apache",
						Repo:    "https://suse-edge.github.io/charts",
						Version: "",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm Chart 'version' field must be defined.",
			},
		},
		`create namespace no target`: {
			K8s: image.Kubernetes{
				HelmCharts: []image.HelmChart{
					{
						Name:            "apache",
						Repo:            "https://suse-edge.github.io/charts",
						Version:         "0.13.10",
						CreateNamespace: true,
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm Chart 'createNamespace' field cannot be true without 'targetNamespace' being defined.",
			},
		},
		`invalid values file`: {
			K8s: image.Kubernetes{
				HelmCharts: []image.HelmChart{
					{
						Name:       "apache",
						Repo:       "https://suse-edge.github.io/charts",
						Version:    "0.13.10",
						ValuesFile: "invalid",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm Chart 'valuesFile' field must be the name of a valid yaml file ending in '.yaml' or '.yml'.",
			},
		},
		`nonexistent values file`: {
			K8s: image.Kubernetes{
				HelmCharts: []image.HelmChart{
					{
						Name:       "apache",
						Repo:       "https://suse-edge.github.io/charts",
						Version:    "0.13.10",
						ValuesFile: "nonexistent.yaml",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm Chart Values File 'nonexistent.yaml' could not be found at 'kubernetes/helm/values/nonexistent.yaml'.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			k := test.K8s
			failures := validateHelmCharts(&k, "")
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}
