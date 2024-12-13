package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

var validNetwork = image.Network{
	APIHost: "host.com",
	APIVIP4: "192.168.100.1",
}

func TestValidateKubernetes(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	valuesDir := filepath.Join(configDir, "kubernetes", "helm", "values")
	require.NoError(t, os.MkdirAll(valuesDir, os.ModePerm))

	apacheValuesPath := filepath.Join(valuesDir, "apache-values.yaml")
	require.NoError(t, os.WriteFile(apacheValuesPath, []byte(""), 0o600))

	tests := map[string]struct {
		K8s                    image.Kubernetes
		ExpectedFailedMessages []string
	}{
		`not defined`: {
			K8s: image.Kubernetes{},
		},
		`all valid`: {
			K8s: image.Kubernetes{
				Version: "v1.30.3+k3s1",
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
				Version: "v1.30.3",
				Network: image.Network{
					APIHost: "host.com",
					APIVIP4: "127.0.0.1",
					APIVIP6: "ff02::1",
				},
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
				"Non-unicast cluster API address (127.0.0.1) for field 'apiVIP' is invalid.",
				"Non-unicast cluster API address (127.0.0.1) for field 'apiVIP' is invalid.",
				fmt.Sprintf("Kubernetes server config could not be found at '%s'; dual-stack configuration requires a valid cluster-cidr and service-cidr.", filepath.Join(configDir, "kubernetes", "config", "server.yaml")),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := image.Context{
				ImageConfigDir: configDir,
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
		Version: "v1.30.3+k3s1",
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
			failures := validateHelm(&k, "kubernetes/helm/values", "kubernetes/helm/certs")
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

func TestValidateAdditionalArtifacts(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	manifestsDir := filepath.Join(configDir, "kubernetes", "manifests")
	require.NoError(t, os.MkdirAll(manifestsDir, os.ModePerm))

	testManifest := filepath.Join(manifestsDir, "manifest.yaml")
	require.NoError(t, os.WriteFile(testManifest, []byte(""), 0o600))

	tests := map[string]struct {
		K8s                    image.Kubernetes
		ExpectedFailedMessages []string
	}{
		`missing versions all sections`: {
			K8s: image.Kubernetes{
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
				"Kubernetes version must be defined when Helm charts are specified",
				"Kubernetes version must be defined when manifest URLs are specified",
				"Kubernetes version must be defined when local manifests are configured",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := &image.Context{
				ImageConfigDir: configDir,
				ImageDefinition: &image.Definition{
					Kubernetes: test.K8s,
				},
			}
			failures := validateAdditionalArtifacts(ctx)
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

func TestValidateNetwork(t *testing.T) {
	tests := map[string]struct {
		K8s                    image.Kubernetes
		ExpectedFailedMessages []string
	}{
		`no network defined, no nodes defined`: {
			K8s: image.Kubernetes{
				Network: image.Network{},
			},
		},
		`no network defined, nodes defined`: {
			K8s: image.Kubernetes{
				Network: image.Network{},
				Nodes: []image.Node{
					{
						Hostname:    "node1",
						Type:        "server",
						Initialiser: false,
					},
					{
						Hostname:    "node2",
						Type:        "server",
						Initialiser: false,
					},
				},
			},
			ExpectedFailedMessages: []string{
				"At least one of the (`apiVIP`, `apiVIP6`) fields is required in the 'network' section for multi node clusters.",
			},
		},
		`valid IPv4`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP4: "192.168.1.1",
				},
			},
		},
		`invalid IPv4`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP4: "500.168.1.1",
				},
			},
			ExpectedFailedMessages: []string{
				"Invalid address value \"500.168.1.1\" for field 'apiVIP'.",
			},
		},
		`valid IPv6`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP6: "fd12:3456:789a::21",
				},
			},
		},
		`invalid IPv6`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP6: "xxxx:3456:789a::21",
				},
			},
			ExpectedFailedMessages: []string{
				"Invalid address value \"xxxx:3456:789a::21\" for field 'apiVIP6'.",
			},
		},
		`valid dualstack`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP4: "192.168.1.1",
					APIVIP6: "fd12:3456:789a::21",
				},
			},
		},
		`invalid dualstack IPv4 non unicast`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP4: "127.0.0.1",
					APIVIP6: "fd12:3456:789a::21",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (127.0.0.1) for field 'apiVIP' is invalid.",
			},
		},
		`invalid dualstack IPv6 non unicast`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP4: "192.168.1.1",
					APIVIP6: "ff02::1",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (ff02::1) for field 'apiVIP6' is invalid.",
			},
		},
		`invalid dualstack both non unicast`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP4: "127.0.0.1",
					APIVIP6: "ff02::1",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (127.0.0.1) for field 'apiVIP' is invalid.",
				"Non-unicast cluster API address (ff02::1) for field 'apiVIP6' is invalid.",
			},
		},
		`invalid dualstack IPv4 not valid`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP4: "500.168.1.1",
					APIVIP6: "fd12:3456:789a::21",
				},
			},
			ExpectedFailedMessages: []string{
				"Invalid address value \"500.168.1.1\" for field 'apiVIP'.",
			},
		},
		`invalid dualstack IPv6 not valid`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIVIP4: "192.168.1.1",
					APIVIP6: "xxxx:3456:789a::21",
				},
			},
			ExpectedFailedMessages: []string{
				"Invalid address value \"xxxx:3456:789a::21\" for field 'apiVIP6'.",
			},
		},
		`undefined v4 VIP`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIHost: "host.com",
					APIVIP4: "0.0.0.0",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (0.0.0.0) for field 'apiVIP' is invalid.",
			},
		},
		`undefined v6 VIP`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIHost: "host.com",
					APIVIP6: "::",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (::) for field 'apiVIP6' is invalid.",
			},
		},
		`loopback v4 VIP`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIHost: "host.com",
					APIVIP4: "127.0.0.1",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (127.0.0.1) for field 'apiVIP' is invalid.",
			},
		},
		`loopback v6 VIP`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIHost: "host.com",
					APIVIP6: "::1",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (::1) for field 'apiVIP6' is invalid.",
			},
		},
		`multicast v4 VIP`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIHost: "host.com",
					APIVIP4: "224.224.224.224",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (224.224.224.224) for field 'apiVIP' is invalid.",
			},
		},
		`multicast v6 VIP`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIHost: "host.com",
					APIVIP6: "FF01::1",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (FF01::1) for field 'apiVIP6' is invalid.",
			},
		},
		`link-local v4 VIP`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIHost: "host.com",
					APIVIP4: "169.254.1.1",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (169.254.1.1) for field 'apiVIP' is invalid.",
			},
		},
		`link-local v6 VIP`: {
			K8s: image.Kubernetes{
				Network: image.Network{
					APIHost: "host.com",
					APIVIP6: "FE80::1",
				},
			},
			ExpectedFailedMessages: []string{
				"Non-unicast cluster API address (FE80::1) for field 'apiVIP6' is invalid.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			k := test.K8s
			failures := validateNetwork(&k)
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

func TestValidateConfigValidIPv6Prio(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	serverConfigDir := filepath.Join(configDir, "kubernetes", "config")
	require.NoError(t, os.MkdirAll(serverConfigDir, os.ModePerm))

	serverConfig := map[string]any{
		"cluster-cidr": "fd12:3456:789b::/48,10.42.0.0/16",
		"service-cidr": "fd12:3456:789c::/112,10.43.0.0/16",
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configFile := filepath.Join(serverConfigDir, "server.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(b), 0o600))

	failures := validateNetworkingConfig(&k8s, configFile)

	assert.Len(t, failures, 0)
}

func TestValidateConfigValidIPv4Prio(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	serverConfigDir := filepath.Join(configDir, "kubernetes", "config")
	require.NoError(t, os.MkdirAll(serverConfigDir, os.ModePerm))

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16,fd12:3456:789b::/48",
		"service-cidr": "10.43.0.0/16,fd12:3456:789c::/112",
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configFile := filepath.Join(serverConfigDir, "server.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(b), 0o600))

	failures := validateNetworkingConfig(&k8s, configFile)

	assert.Len(t, failures, 0)
}

func TestValidateConfigInvalidBothIPv4(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	serverConfigDir := filepath.Join(configDir, "kubernetes", "config")
	require.NoError(t, os.MkdirAll(serverConfigDir, os.ModePerm))

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16,10.44.0.0/16",
		"service-cidr": "10.43.0.0/16,10.45.0.0/16",
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configFile := filepath.Join(serverConfigDir, "server.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(b), 0o600))

	failures := validateNetworkingConfig(&k8s, configFile)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr cannot contain addresses of the same IP address family; one must be IPv4, and the other IPv6")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr cannot contain addresses of the same IP address family; one must be IPv4, and the other IPv6")
}

func TestValidateConfigInvalidBothIPv6(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	serverConfigDir := filepath.Join(configDir, "kubernetes", "config")
	require.NoError(t, os.MkdirAll(serverConfigDir, os.ModePerm))

	serverConfig := map[string]any{
		"cluster-cidr": "fd12:3456:789d::/48,fd12:3456:789b::/48",
		"service-cidr": "fd12:3456:789e::/112,fd12:3456:789c::/112",
	}

	b, err := yaml.Marshal(serverConfig)
	require.NoError(t, err)

	configFile := filepath.Join(serverConfigDir, "server.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(b), 0o600))

	failures := validateNetworkingConfig(&k8s, configFile)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr cannot contain addresses of the same IP address family; one must be IPv4, and the other IPv6")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr cannot contain addresses of the same IP address family; one must be IPv4, and the other IPv6")
}

func TestValidateConfigInvalidServerConfigNotConfigured(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	failures := validateNetworkingConfig(&k8s, "fake-path")

	assert.Len(t, failures, 1)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config could not be found at 'fake-path'; dual-stack configuration requires a valid cluster-cidr and service-cidr.")
}

func TestValidateConfigValidAPIVIPNotConfigured(t *testing.T) {
	k8s := image.Kubernetes{}

	failures := validateNetworkingConfig(&k8s, "")
	assert.Len(t, failures, 0)
}

func TestValidateConfigInvalidClusterCIDRNotConfigured(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	serverConfig := map[string]any{
		"service-cidr": "10.43.0.0/16,fd12:3456:789c::/112",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 1)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config must contain a valid cluster-cidr when configuring dual-stack")
}

func TestValidateConfigInvalidServiceCIDRNotConfigured(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	serverConfig := map[string]any{
		"cluster-cidr": "fd12:3456:789b::/48,10.42.0.0/16",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 1)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config must contain a valid service-cidr when configuring dual-stack")
}

func TestValidateConfigInvalidIPv4(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}
	serverConfig := map[string]any{
		"cluster-cidr": "500.42.0.0/16,fd12:3456:789b::/48",
		"service-cidr": "500.43.0.0/16,fd12:3456:789c::/112",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr value '500.42.0.0/16' could not be parsed")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr value '500.43.0.0/16' could not be parsed")
}

func TestValidateConfigInvalidIPv6(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16,xxxx:3456:789b::/48",
		"service-cidr": "10.43.0.0/16,xxxx:3456:789c::/112",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr value 'xxxx:3456:789b::/48' could not be parsed")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr value 'xxxx:3456:789c::/112' could not be parsed")
}

func TestValidateConfigInvalidIPv6Prefix(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16,fd12:3456:789a::/480",
		"service-cidr": "10.43.0.0/16,fd12:3456:789a::/1122",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr value 'fd12:3456:789a::/480' could not be parsed")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr value 'fd12:3456:789a::/1122' could not be parsed")
}

func TestValidateConfigInvalidIPv4Prefix(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/50,fd12:3456:789a::/48",
		"service-cidr": "10.43.0.0/50,fd12:3456:789a::/112",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr value '10.42.0.0/50' could not be parsed")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr value '10.43.0.0/50' could not be parsed")
}

func TestValidateConfigInvalidIPv4NonUnicast(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	serverConfig := map[string]any{
		"cluster-cidr": "127.0.0.1/16,fd12:3456:789a::/48",
		"service-cidr": "127.0.0.1/16,fd12:3456:789a::/112",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr value '127.0.0.1/16' must be a valid unicast address")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr value '127.0.0.1/16' must be a valid unicast address")
}

func TestValidateConfigInvalidIPv6NonUnicast(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16,FF01::/48",
		"service-cidr": "10.43.0.0/16,FF01::/112",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr value 'FF01::/48' must be a valid unicast address")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr value 'FF01::/112' must be a valid unicast address")
}

func TestValidateNodeIPValid(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
		APIVIP6: "fd12:3456:789a::21",
	}}

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16,FF01::/48",
		"service-cidr": "10.43.0.0/16,FF01::/112",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 2)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr value 'FF01::/48' must be a valid unicast address")
	assert.Contains(t, foundMessages, "Kubernetes server config service-cidr value 'FF01::/112' must be a valid unicast address")
}

func TestValidateConfigMismatchedPrio(t *testing.T) {
	k8s := image.Kubernetes{}

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16,fd12:3456:789b::/48",
		"service-cidr": "fd12:3456:789c::/112,10.43.0.0/16",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 1)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config cluster-cidr cannot prioritize one address family while service-cidr prioritizes another; both must have the same priority")
}

func TestValidateConfigSingleCIDRIPv6(t *testing.T) {
	k8s := image.Kubernetes{}

	serverConfig := map[string]any{
		"cluster-cidr": "fd12:3456:789b::/48",
		"service-cidr": "fd12:3456:789c::/112",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 0)
}

func TestValidateConfigSingleCIDRIPv4(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{
		APIVIP4: "192.168.1.1",
	}}

	serverConfig := map[string]any{
		"cluster-cidr": "10.42.0.0/16",
		"service-cidr": "10.43.0.0/16",
	}

	failures := validateCIDRConfig(&k8s, serverConfig)

	assert.Len(t, failures, 0)
}

func TestValidateNodeIPValidSingleIPv4(t *testing.T) {
	k8s := image.Kubernetes{}

	serverConfig := map[string]any{
		"node-ip": "10.42.0.0",
	}

	failures := validateNodeIP(&k8s, serverConfig)

	assert.Len(t, failures, 0)
}

func TestValidateNodeIPValidSingleIPv6(t *testing.T) {
	k8s := image.Kubernetes{Network: image.Network{}}

	serverConfig := map[string]any{
		"node-ip": "fd12:3456:789a::21",
	}

	failures := validateNodeIP(&k8s, serverConfig)

	assert.Len(t, failures, 0)
}

func TestValidateNodeIPMultipleNodesValid(t *testing.T) {
	k8s := image.Kubernetes{Nodes: []image.Node{
		{
			Hostname: "agent1",
			Type:     image.KubernetesNodeTypeAgent,
		},
		{
			Hostname: "agent2",
			Type:     image.KubernetesNodeTypeAgent,
		},
		{
			Hostname:    "server",
			Type:        image.KubernetesNodeTypeServer,
			Initialiser: true,
		},
	}}

	serverConfig := map[string]any{
		"node-ip": "10.42.0.0",
	}

	failures := validateNodeIP(&k8s, serverConfig)

	assert.Len(t, failures, 0)
}

func TestValidateNodeIPMultipleServersInvalid(t *testing.T) {
	k8s := image.Kubernetes{Nodes: []image.Node{
		{
			Hostname: "server1",
			Type:     image.KubernetesNodeTypeServer,
		},
		{
			Hostname:    "server2",
			Type:        image.KubernetesNodeTypeServer,
			Initialiser: true,
		},
	}}

	serverConfig := map[string]any{
		"node-ip": "10.42.0.0",
	}

	failures := validateNodeIP(&k8s, serverConfig)

	assert.Len(t, failures, 1)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config node-ip can not be specified when there is more than one Kubernetes server node")
}

func TestValidateNodeIPBothSameFamilyInvalid(t *testing.T) {
	k8s := image.Kubernetes{}

	serverConfig := map[string]any{
		"node-ip": "10.42.0.0,10.43.0.0",
	}

	failures := validateNodeIP(&k8s, serverConfig)

	assert.Len(t, failures, 1)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config node-ip cannot contain addresses of the same IP address family; one must be IPv4, and the other IPv6")
}

func TestValidateNodeIPDualstackValid(t *testing.T) {
	k8s := image.Kubernetes{}

	serverConfig := map[string]any{
		"node-ip": "10.42.0.0,fd12:3456:789a::21",
	}

	failures := validateNodeIP(&k8s, serverConfig)

	assert.Len(t, failures, 0)
}

func TestValidateNodeIPNonunicastIPv4Invalid(t *testing.T) {
	k8s := image.Kubernetes{}

	serverConfig := map[string]any{
		"node-ip": "127.0.0.1,fd12:3456:789a::21",
	}

	failures := validateNodeIP(&k8s, serverConfig)

	assert.Len(t, failures, 1)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config node-ip value '127.0.0.1' must be a valid unicast address")
}

func TestValidateNodeIPIPv6Invalid(t *testing.T) {
	k8s := image.Kubernetes{}

	serverConfig := map[string]any{
		"node-ip": "xxxx:3456:789a::21",
	}

	failures := validateNodeIP(&k8s, serverConfig)

	assert.Len(t, failures, 1)

	var foundMessages []string
	for _, foundValidation := range failures {
		foundMessages = append(foundMessages, foundValidation.UserMessage)
	}

	assert.Contains(t, foundMessages, "Kubernetes server config node-ip value 'xxxx:3456:789a::21' could not be parsed")
}
