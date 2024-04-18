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
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:                  "apache",
							RepositoryName:        "apache-repo",
							TargetNamespace:       "web",
							CreateNamespace:       true,
							InstallationNamespace: "kube-system",
							Version:               "10.7.0",
							ValuesFile:            "apache-values.yaml",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
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
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "",
							RepositoryName: "another-apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'hostname' field is required for entries in the 'nodes' section.",
				"Entries in 'urls' must begin with either 'http://' or 'https://'.",
				"Helm chart 'name' field must be defined.",
				"Helm repository 'name' field for \"apache-repo\" must match the 'repositoryName' field in at least one defined Helm chart.",
				"Helm chart 'repositoryName' \"another-apache-repo\" for Helm chart \"\" does not match the name of any defined repository.",
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
		Network:   image.Network{},
		Nodes:     []image.Node{},
		Manifests: image.Manifests{},
		Helm:      image.Helm{},
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
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:                  "apache",
							RepositoryName:        "apache-repo",
							TargetNamespace:       "web",
							CreateNamespace:       true,
							InstallationNamespace: "kube-system",
							Version:               "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
							Authentication: image.HelmAuthentication{
								Username: "user",
								Password: "pass",
							},
						},
					},
				},
			},
		},
		`helm no repos`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm charts defined with no Helm repositories defined.",
			},
		},
		`helm chart no name`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm chart 'name' field must be defined.",
			},
		},
		`helm chart undefined repository name`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "kubevirt",
							RepositoryName: "suse-edge",
							Version:        "0.2.2",
						},
						{
							Name:           "metallb",
							RepositoryName: "",
							Version:        "0.14.3",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "suse-edge",
							URL:  "https://suse-edge.github.io/charts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm chart 'repositoryName' field for \"metallb\" must be defined.",
			},
		},
		`helm chart no matching repository name`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "kubevirt",
							RepositoryName: "suse-edge",
							Version:        "0.2.2",
						},
						{
							Name:           "metallb",
							RepositoryName: "this-is-not-suse-edge",
							Version:        "0.14.3",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "suse-edge",
							URL:  "https://suse-edge.github.io/charts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm chart 'repositoryName' \"this-is-not-suse-edge\" for Helm chart \"metallb\" does not match the name of any defined repository.",
			},
		},
		`helm chart no version`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm chart 'version' field for \"apache\" field must be defined.",
			},
		},
		`helm chart create namespace no target`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:            "apache",
							RepositoryName:  "apache-repo",
							Version:         "10.7.0",
							CreateNamespace: true,
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm chart 'createNamespace' field for \"apache\" cannot be true without 'targetNamespace' being defined.",
			},
		},
		`helm chart duplicate name`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'helmCharts' field contains duplicate entries: apache",
			},
		},
		`helm chart invalid values file`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
							ValuesFile:     "invalid",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm chart 'valuesFile' field for \"apache\" must be the name of a valid yaml file ending in '.yaml' or '.yml'.",
			},
		},
		`helm chart nonexistent values file`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
							ValuesFile:     "nonexistent.yaml",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm chart values file 'nonexistent.yaml' could not be found at 'kubernetes/helm/values/nonexistent.yaml'.",
			},
		},
		`helm repository no name`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "",
							URL:  "https://suse-edge.github.io/charts",
						},
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'name' field must be defined.",
			},
		},
		`helm repository no url`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'url' field for \"apache-repo\" must be defined.",
			},
		},
		`helm repository invalid url`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "invalid.repo.io/bitnami",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'url' field for \"apache-repo\" must begin with either 'oci://', 'http://', or 'https://'.",
			},
		},
		`helm repository username no password`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
							Authentication: image.HelmAuthentication{
								Username: "user",
								Password: "",
							},
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'password' field not defined for \"apache-repo\".",
			},
		},
		`helm repository password no username`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name: "apache-repo",
							URL:  "oci://registry-1.docker.io/bitnamicharts",
							Authentication: image.HelmAuthentication{
								Username: "",
								Password: "pass",
							},
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'username' field not defined for \"apache-repo\".",
			},
		},
		`helm repository both skipTLSVerify and plainHTTP true`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name:          "apache-repo",
							URL:           "oci://registry-1.docker.io/bitnamicharts",
							SkipTLSVerify: true,
							PlainHTTP:     true,
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'plainHTTP' and 'skipTLSVerify' fields for \"apache-repo\" cannot both be true.",
			},
		},
		`helm repository skipTLSVerify true for http`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "metallb",
							RepositoryName: "suse-edge",
							Version:        "0.14.3",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name:          "suse-edge",
							URL:           "http://suse-edge.github.io/charts",
							SkipTLSVerify: true,
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'url' field for \"suse-edge\" contains 'http://' but 'plainHTTP' field is false.",
				"Helm repository 'url' field for \"suse-edge\" contains 'http://' but 'skipTLSVerify' field is true.",
			},
		},
		`helm repository plainHTTP false for http`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "metallb",
							RepositoryName: "suse-edge",
							Version:        "0.14.3",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name:      "suse-edge",
							URL:       "http://suse-edge.github.io/charts",
							PlainHTTP: false,
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'url' field for \"suse-edge\" contains 'http://' but 'plainHTTP' field is false.",
			},
		},
		`helm repository plainHTTP true for https`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "metallb",
							RepositoryName: "suse-edge",
							Version:        "0.14.3",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name:      "suse-edge",
							URL:       "https://suse-edge.github.io/charts",
							PlainHTTP: true,
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'url' field for \"suse-edge\" contains 'https://' but 'plainHTTP' field is true.",
			},
		},
		`helm repository plainHTTP and ca file`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "metallb",
							RepositoryName: "suse-edge",
							Version:        "0.14.3",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name:      "suse-edge",
							URL:       "http://suse-edge.github.io/charts",
							PlainHTTP: true,
							CAFile:    "suse-edge.crt",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'caFile' field for \"suse-edge\" cannot be defined while 'plainHTTP' is true.",
				"Helm repository 'url' field for \"suse-edge\" contains 'http://' but 'caFile' field is defined.",
				"Helm repo cert file/bundle 'suse-edge.crt' could not be found at 'kubernetes/helm/certs/suse-edge.crt'.",
			},
		},
		`helm repository skipTLSVerify and ca file`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "metallb",
							RepositoryName: "suse-edge",
							Version:        "0.14.3",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name:          "suse-edge",
							URL:           "https://suse-edge.github.io/charts",
							SkipTLSVerify: true,
							CAFile:        "suse-edge.crt",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repository 'caFile' field for \"suse-edge\" cannot be defined while 'skipTLSVerify' is true.",
				"Helm repo cert file/bundle 'suse-edge.crt' could not be found at 'kubernetes/helm/certs/suse-edge.crt'.",
			},
		},
		`helm repo nonexistent cert file`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name:   "apache-repo",
							URL:    "oci://registry-1.docker.io/bitnamicharts",
							CAFile: "nonexistent-apache.crt",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm repo cert file/bundle 'nonexistent-apache.crt' could not be found at 'kubernetes/helm/certs/nonexistent-apache.crt'.",
			},
		},
		`helm repo invalid cert file`: {
			K8s: image.Kubernetes{
				Helm: image.Helm{
					Charts: []image.HelmChart{
						{
							Name:           "apache",
							RepositoryName: "apache-repo",
							Version:        "10.7.0",
						},
					},
					Repositories: []image.HelmRepository{
						{
							Name:   "apache-repo",
							URL:    "oci://registry-1.docker.io/bitnamicharts",
							CAFile: "invalid-cert",
						},
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Helm chart 'caFile' field for \"apache-repo\" must be the name of a valid cert file/bundle with one of the " +
					"following extensions: .pem, .crt, .cer",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			k := test.K8s
			failures := validateHelm(&k, "")
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
