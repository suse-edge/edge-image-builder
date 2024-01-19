package image

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDefinition(t *testing.T) {
	// Setup
	filename := "./testdata/full-valid-example.yaml"
	configData, err := os.ReadFile(filename)
	require.NoError(t, err)
	definition, err := ParseDefinition(configData)
	require.NoError(t, err)

	// Test
	assert.NoError(t, ValidateDefinition(definition))
}

func TestValidateImageValid(t *testing.T) {
	// Setup
	def := Definition{
		Image: Image{
			ImageType:       "iso",
			Arch:            "x86_64",
			BaseImage:       "baseimage.iso",
			OutputImageName: "output.iso",
		},
	}

	// Test
	err := validateImage(&def)

	// Verify
	require.NoError(t, err)
}

func TestValidateImageInvalidImageType(t *testing.T) {
	// Setup
	def := Definition{
		Image: Image{
			ImageType:       "random",
			BaseImage:       "baseimage.iso",
			OutputImageName: "output.iso",
		},
	}

	// Test
	err := validateImage(&def)

	// Verify
	assert.EqualError(t, err, "imageType must be 'iso' or 'raw'")
}

func TestValidateImageUndefinedImageType(t *testing.T) {
	// Setup
	def := Definition{
		Image: Image{
			ImageType:       "",
			BaseImage:       "baseimage.iso",
			OutputImageName: "output.iso",
		},
	}

	// Test
	err := validateImage(&def)

	// Verify
	require.ErrorContains(t, err, "imageType not defined")
}

func TestValidateKubernetes(t *testing.T) {
	tests := []struct {
		name        string
		definition  *Definition
		expectedErr string
	}{
		{
			name: "Valid single node",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
				},
			},
		},
		{
			name: "Multi node empty hostname",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Type:     "server",
							Hostname: "node1.suse.com",
						},
						{
							Type: "server",
						},
					},
				},
			},
			expectedErr: "validating nodes: node hostname cannot be empty",
		},
		{
			name: "Multi node invalid type",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Type:     "server",
							Hostname: "node1.suse.com",
						},
						{
							Type:     "worker",
							Hostname: "node2.suse.com",
						},
					},
				},
			},
			expectedErr: "validating nodes: invalid node type: worker",
		},
		{
			name: "Multi node empty VIP",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Nodes: []Node{
						{
							Type:     "server",
							Hostname: "node1.suse.com",
						},
						{
							Type:     "agent",
							Hostname: "node2.suse.com",
						},
					},
				},
			},
			expectedErr: "validating nodes: virtual API address is not provided",
		},
		{
			name: "Multi node empty API host",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP: "192.168.0.1",
					},
					Nodes: []Node{
						{
							Type:     "server",
							Hostname: "node1.suse.com",
						},
						{
							Type:     "agent",
							Hostname: "node2.suse.com",
						},
					},
				},
			},
			expectedErr: "validating nodes: API host is not provided",
		},
		{
			name: "Multi node duplicate node hostnames",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Type:     "server",
							Hostname: "node1.suse.com",
						},
						{
							Type:     "server",
							Hostname: "node1.suse.com",
						},
					},
				},
			},
			expectedErr: "validating nodes: node list contains duplicate: node1.suse.com",
		},
		{
			name: "Multi node agents only",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Type:     "agent",
							Hostname: "node1.suse.com",
						},
						{
							Type:     "agent",
							Hostname: "node2.suse.com",
						},
					},
				},
			},
			expectedErr: "validating nodes: cluster of only agent nodes cannot be formed",
		},
		{
			name: "Multi node agent initialiser",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Type:     "server",
							Hostname: "node1.suse.com",
						},
						{
							Type:     "agent",
							Hostname: "node2.suse.com",
							First:    true,
						},
					},
				},
			},
			expectedErr: "validating nodes: agent nodes cannot be cluster initialisers: node2.suse.com",
		},
		{
			name: "Multi node multiple initialisers",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Type:     "server",
							Hostname: "node1.suse.com",
							First:    true,
						},
						{
							Type:     "server",
							Hostname: "node2.suse.com",
							First:    true,
						},
					},
				},
			},
			expectedErr: "validating nodes: only one node can be cluster initialiser",
		},
		{
			name: "Valid multi node",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Type:     "server",
							Hostname: "node1.suse.com",
							First:    true,
						},
						{
							Type:     "agent",
							Hostname: "node2.suse.com",
						},
					},
				},
			},
		},
		{
			name: "Valid manifest URLs",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Hostname: "node1.suse.com",
							Type:     "server",
						},
					},
					Manifests: Manifests{
						URLs: []string{
							"https://k8s.io/examples/application/nginx-app.yaml",
							"http://localhost:5000/manifest.yaml",
						},
					},
				},
			},
		},
		{
			name: "Duplicate manifest URLs",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Hostname: "node1.suse.com",
							Type:     "server",
						},
					},
					Manifests: Manifests{
						URLs: []string{
							"https://k8s.io/examples/application/nginx-app.yaml",
							"https://k8s.io/examples/application/nginx-app.yaml",
						},
					},
				},
			},
			expectedErr: "validating manifest urls: duplicate manifest url found: 'https://k8s.io/examples/application/nginx-app.yaml'",
		},
		{
			name: "Invalid manifest URL",
			definition: &Definition{
				Kubernetes: Kubernetes{
					Version: "v1.29.0+rke2r1",
					Network: Network{
						APIVIP:  "192.168.0.1",
						APIHost: "api.cluster01.hosted.on.edge.suse.com",
					},
					Nodes: []Node{
						{
							Hostname: "node1.suse.com",
							Type:     "server",
						},
					},
					Manifests: Manifests{
						URLs: []string{
							"k8s.io/examples/application/nginx-app.yaml",
						},
					},
				},
			},
			expectedErr: "validating manifest urls: invalid manifest url, does not start with 'http://' or 'https://'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateKubernetes(test.definition)

			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedErr)
			}
		})
	}
}

func TestValidateImage_Arch(t *testing.T) {
	tests := []struct {
		name        string
		definition  *Definition
		expectedErr string
	}{
		{
			name: "Undefined arch",
			definition: &Definition{
				Image: Image{
					ImageType: "iso",
					Arch:      "",
				},
			},
			expectedErr: "arch not defined",
		},
		{
			name: "Invalid arch",
			definition: &Definition{
				Image: Image{
					ImageType: "iso",
					Arch:      "arm64",
				},
			},
			expectedErr: "arch must be 'x86_64' or 'aarch64'",
		},
		{
			name: "Valid AMD arch",
			definition: &Definition{
				Image: Image{
					ImageType:       "iso",
					Arch:            "x86_64",
					BaseImage:       "img.iso",
					OutputImageName: "out.iso",
				},
			},
		},
		{
			name: "Valid ARM arch",
			definition: &Definition{
				Image: Image{
					ImageType:       "raw",
					Arch:            "aarch64",
					BaseImage:       "img.raw",
					OutputImageName: "out.raw",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateImage(test.definition)

			if test.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedErr)
			}
		})
	}
}

func TestValidateImageUndefinedBaseImage(t *testing.T) {
	// Setup
	def := Definition{
		Image: Image{
			ImageType:       "raw",
			Arch:            "x86_64",
			BaseImage:       "",
			OutputImageName: "output.iso",
		},
	}

	// Test
	err := validateImage(&def)

	// Verify
	require.ErrorContains(t, err, "baseImage not defined")
}

func TestValidateImageUndefinedOutputImageName(t *testing.T) {
	// Setup
	def := Definition{
		Image: Image{
			ImageType:       "raw",
			Arch:            "x86_64",
			BaseImage:       "baseimage.iso",
			OutputImageName: "",
		},
	}

	// Test
	err := validateImage(&def)

	// Verify
	require.ErrorContains(t, err, "outputImageName not defined")
}

func TestValidateOperatingSystemEmptyButValid(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{},
	}

	// Test
	err := validateOperatingSystem(&def)

	// Verify
	require.NoError(t, err)
}

func TestValidateOperatingSystemValid(t *testing.T) {
	// Setup
	def := Definition{
		Image: Image{
			ImageType: "iso",
		},
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key1=value1", "key2=value2", "arg1", "arg2"},
			Systemd: Systemd{
				Enable:  []string{"enable1", "enable2", "enable3"},
				Disable: []string{"disable1", "disable2", "disable3"},
			},
			Users: []OperatingSystemUser{
				{
					Username:          "user1",
					EncryptedPassword: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:            "ssh-rsa AAAqeCzFPRrNyA5a",
				},
				{
					Username:          "user2",
					EncryptedPassword: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:            "ssh-rsa AAAqeCzFPRrNyA5a",
				},
			},
			Suma: Suma{
				Host:          "suma.edge.suse.com",
				ActivationKey: "slemicro55",
				GetSSL:        false,
			},
			InstallDevice: "/dev/sda",
			Unattended:    true,
		},
	}

	// Test
	err := validateOperatingSystem(&def)

	// Verify
	require.NoError(t, err)
}

func TestValidateOperatingSystemValidKernelArgs(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key1=value1", "key2=value2", "arg1", "arg2"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.NoError(t, err)
}

func TestValidateOperatingSystemKernelArgMissingKey(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key1=value1", "=value2"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "has no key")
}

func TestValidateOperatingSystemKernelArgMissingValue(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key1=", "key2=value2"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "has no value")
}

func TestValidateOperatingSystemKernelArgMixedFormats(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"arg1", "key2=value2"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.NoError(t, err)
}

func TestValidateOperatingSystemKernelArgDuplicatesInMixedFormat(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key2", "key2=value2"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "duplicate kernel arg found")
}

func TestValidateOperatingSystemKernelArgDuplicateArgs(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key1=value2", "key1=value2"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "duplicate kernel arg")
}

func TestValidateOperatingSystemKernelArgDuplicateArgsSecondFormat(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key1", "key1"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "duplicate kernel arg")
}

func TestValidateOperatingSystemSystemdValid(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Systemd: Systemd{
				Enable:  []string{"enable0", "enable1"},
				Disable: []string{"disable0", "disable1"},
			},
		},
	}

	// Test
	err := validateSystemd(&def.OperatingSystem)

	// Verify
	require.NoError(t, err)
}

func TestValidateOperatingSystemSystemdEnableListDuplicate(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Systemd: Systemd{
				Enable:  []string{"enable0", "enable0"},
				Disable: []string{"disable0", "disable1"},
			},
		},
	}

	// Test
	err := validateSystemd(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "enable list contains duplicate")
}

func TestValidateOperatingSystemSystemdDisableListDuplicate(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Systemd: Systemd{
				Enable:  []string{"enable0", "enable1"},
				Disable: []string{"disable0", "disable0"},
			},
		},
	}

	// Test
	err := validateSystemd(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "disable list contains duplicate")
}

func TestValidateOperatingSystemSystemdListConflicts(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Systemd: Systemd{
				Enable:  []string{"enable0", "enable1"},
				Disable: []string{"enable0", "disable1"},
			},
		},
	}

	// Test
	err := validateSystemd(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "conflict found")
}

func TestValidateOperatingSystemUsersValid(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Users: []OperatingSystemUser{
				{
					Username:          "user1",
					EncryptedPassword: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:            "ssh-rsa AAAqeCzFPRrNyA5a",
				},
			},
		},
	}

	// Test
	err := validateUsers(&def.OperatingSystem)

	// Verify
	require.NoError(t, err)
}

func TestValidateOperatingSystemUsersMissingUsername(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Users: []OperatingSystemUser{
				{
					Username:          "",
					EncryptedPassword: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:            "ssh-rsa AAAqeCzFPRrNyA5a",
				},
			},
		},
	}

	// Test
	err := validateUsers(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "missing username")
}

func TestValidateOperatingSystemUsersDuplicateUsername(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Users: []OperatingSystemUser{
				{
					Username:          "user1",
					EncryptedPassword: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:            "ssh-rsa AAAqeCzFPRrNyA5a",
				},
				{
					Username:          "user1",
					EncryptedPassword: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:            "ssh-rsa AAAqeCzFPRrNyA5a",
				},
			},
		},
	}

	// Test
	err := validateUsers(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "duplicate username")
}

func TestValidateOperatingSystemUsersNoSSHKeyOrPassword(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Users: []OperatingSystemUser{
				{
					Username:          "user1",
					EncryptedPassword: "",
					SSHKey:            "",
				},
			},
		},
	}

	// Test
	err := validateUsers(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "must have either a password or an SSH key")
}

func TestValidateOperatingSystemSumaValid(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Suma: Suma{
				Host:          "suma.edge.suse.com",
				ActivationKey: "slemicro55",
				GetSSL:        false,
			},
		},
	}

	// Test
	err := validateSuma(&def.OperatingSystem)

	// Verify
	require.NoError(t, err)
}

func TestValidateOperatingSystemSumaEmptyButValid(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Suma: Suma{
				Host:          "",
				ActivationKey: "",
				GetSSL:        false,
			},
		},
	}

	// Test
	err := validateSuma(&def.OperatingSystem)

	// Verify
	require.NoError(t, err)
}

func TestValidateOperatingSystemSumaMissingHost(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Suma: Suma{
				Host:          "",
				ActivationKey: "slemicro55",
				GetSSL:        false,
			},
		},
	}

	// Test
	err := validateSuma(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "no host defined")
}

func TestValidateOperatingSystemSumaInvalidHostHTTP(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Suma: Suma{
				Host:          "http://hostname",
				ActivationKey: "slemicro55",
				GetSSL:        false,
			},
		},
	}

	// Test
	err := validateSuma(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "invalid hostname, hostname should not contain 'http://'")
}

func TestValidateOperatingSystemSumaInvalidHostHTTPS(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Suma: Suma{
				Host:          "https://hostname",
				ActivationKey: "slemicro55",
				GetSSL:        false,
			},
		},
	}

	// Test
	err := validateSuma(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "invalid hostname, hostname should not contain 'http://'")
}

func TestValidateOperatingSystemSumaMissingActivationKey(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			Suma: Suma{
				Host:          "suma.edge.suse.com",
				ActivationKey: "",
				GetSSL:        false,
			},
		},
	}

	// Test
	err := validateSuma(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "no activation key defined")
}

func TestValidateEmbeddedArtifactRegistry(t *testing.T) {
	// Setup
	def := Definition{EmbeddedArtifactRegistry: EmbeddedArtifactRegistry{
		ContainerImages: []ContainerImage{
			{
				Name: "hello-world:latest",
			},
			{
				Name:           "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
				SupplyChainKey: "carbide-key.pub",
			},
		},
		HelmCharts: []HelmChart{
			{
				Name:    "rancher",
				RepoURL: "https://releases.rancher.com/server-charts/stable",
				Version: "2.8.0",
			},
		},
	}}

	// Test
	err := validateEmbeddedArtifactRegistry(&def)

	// Verify
	require.NoError(t, err)
}

func TestValidateContainerImages(t *testing.T) {
	tests := []struct {
		name        string
		images      []ContainerImage
		expectedErr string
	}{
		{
			name: "Valid Images",
			images: []ContainerImage{
				{
					Name:           "hello-world:latest",
					SupplyChainKey: "",
				},
				{
					Name:           "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
					SupplyChainKey: "carbide-key.pub",
				},
			},
		},
		{
			name: "No Image Name Defined",
			images: []ContainerImage{
				{
					Name:           "",
					SupplyChainKey: "",
				},
			},
			expectedErr: "no image name defined",
		},
		{
			name: "Duplicate Container Image",
			images: []ContainerImage{
				{
					Name:           "hello-world:latest",
					SupplyChainKey: "",
				},
				{
					Name:           "hello-world:latest",
					SupplyChainKey: "carbide-key.pub",
				},
			},
			expectedErr: "duplicate container image found: 'hello-world:latest'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateContainerImages(test.images)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestValidateCharts(t *testing.T) {
	tests := []struct {
		name        string
		charts      []HelmChart
		expectedErr string
	}{
		{
			name: "Valid Charts",
			charts: []HelmChart{
				{
					Name:    "rancher",
					RepoURL: "https://releases.rancher.com/server-charts/stable",
					Version: "2.8.0",
				},
			},
		},
		{
			name: "No Chart Name Defined",
			charts: []HelmChart{
				{
					Name:    "",
					RepoURL: "https://releases.rancher.com/server-charts/stable",
					Version: "2.8.0",
				},
			},
			expectedErr: "no chart name defined",
		},
		{
			name: "No Chart RepoURL Defined",
			charts: []HelmChart{
				{
					Name:    "rancher",
					RepoURL: "",
					Version: "2.8.0",
				},
			},
			expectedErr: "no chart repository URL defined for 'rancher'",
		},
		{
			name: "No Chart Version Defined",
			charts: []HelmChart{
				{
					Name:    "rancher",
					RepoURL: "https://releases.rancher.com/server-charts/stable",
					Version: "",
				},
			},
			expectedErr: "no chart version defined for 'rancher'",
		},
		{
			name: "Invalid Chart RepoURL",
			charts: []HelmChart{
				{
					Name:    "rancher",
					RepoURL: "releases.rancher.com/server-charts/stable",
					Version: "2.8.0",
				},
			},
			expectedErr: "invalid chart respository url, does not start with 'http://' or 'https://'",
		},
		{
			name: "Duplicate Chart",
			charts: []HelmChart{
				{
					Name:    "rancher",
					RepoURL: "https://releases.rancher.com/server-charts/stable",
					Version: "2.8.0",
				},
				{
					Name:    "rancher",
					RepoURL: "https://releases.rancher.com/server-charts/stable",
					Version: "2.8.0",
				},
			},
			expectedErr: "duplicate chart found: 'rancher'",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateHelmCharts(test.charts)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)
			}
		})
	}
}

func TestIsEmbeddedArtifactRegistryEmpty(t *testing.T) {
	tests := []struct {
		name     string
		registry EmbeddedArtifactRegistry
		isEmpty  bool
	}{
		{
			name: "Both Defined",
			registry: EmbeddedArtifactRegistry{
				HelmCharts: []HelmChart{
					{
						Name:    "rancher",
						RepoURL: "https://releases.rancher.com/server-charts/stable",
						Version: "2.8.0",
					},
				},
				ContainerImages: []ContainerImage{
					{
						Name:           "hello-world:latest",
						SupplyChainKey: "",
					},
				},
			},
			isEmpty: false,
		},
		{
			name: "Chart Defined",
			registry: EmbeddedArtifactRegistry{
				HelmCharts: []HelmChart{
					{
						Name:    "rancher",
						RepoURL: "https://releases.rancher.com/server-charts/stable",
						Version: "2.8.0",
					},
				},
			},
			isEmpty: false,
		},
		{
			name: "Image Defined",
			registry: EmbeddedArtifactRegistry{
				ContainerImages: []ContainerImage{
					{
						Name:           "hello-world:latest",
						SupplyChainKey: "",
					},
				},
			},
			isEmpty: false,
		},
		{
			name:    "None Defined",
			isEmpty: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsEmbeddedArtifactRegistryEmpty(test.registry)
			assert.Equal(t, test.isEmpty, result)
		})
	}
}

func TestValidatePackages(t *testing.T) {
	tests := []struct {
		name        string
		os          *OperatingSystem
		expectedErr string
	}{
		{
			name: "Package list with duplicate",
			os: &OperatingSystem{
				Packages: Packages{
					PKGList: []string{"foo", "bar", "foo"},
				},
			},
			expectedErr: "package list contains duplicate: foo",
		},
		{
			name: "Additional repository with duplicate",
			os: &OperatingSystem{
				Packages: Packages{
					AdditionalRepos: []string{"https://foo.bar", "https://bar.foo", "https://foo.bar"},
				},
			},
			expectedErr: "additional repository list contains duplicate: https://foo.bar",
		},
		{
			name: "Package list defined without registration code or third party repo",
			os: &OperatingSystem{
				Packages: Packages{
					PKGList: []string{"foo", "bar"},
				},
			},
			expectedErr: "package list configured without providing additional repository or registration code",
		},
		{
			name: "Configuring package from PackageHub",
			os: &OperatingSystem{
				Packages: Packages{
					PKGList: []string{"foo", "bar"},
					RegCode: "foo.bar",
				},
			},
		},
		{
			name: "Configuring package from third party repo",
			os: &OperatingSystem{
				Packages: Packages{
					PKGList:         []string{"foo", "bar"},
					AdditionalRepos: []string{"https://foo.bar"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePackages(test.os)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
