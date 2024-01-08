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
	require.ErrorContains(t, err, "invalid imageType")
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

func TestValidateImageUndefinedBaseImage(t *testing.T) {
	// Setup
	def := Definition{
		Image: Image{
			ImageType:       "raw",
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

func TestValidateHauler(t *testing.T) {
	// Setup
	def := Definition{Hauler: Hauler{
		ContainerImages: []ContainerImage{
			{
				Name: "hello-world:latest",
			},
			{
				Name: "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
				Key:  "carbide-key.pub",
			},
		},
		HelmCharts: []HelmChart{
			{
				Name:    "rancher",
				RepoURL: "https://releases.rancher.com/server-charts/stable",
				Version: "2.8.0",
			},
		},
		Files: []File{
			{
				Name: "install.sh",
				Path: "https://get.rke2.io",
			},
		},
	}}

	// Test
	err := validateHauler(&def)

	// Verify
	require.NoError(t, err)
}

func TestValidateContainerImages(t *testing.T) {
	// Setup
	tests := []struct {
		name        string
		images      []ContainerImage
		expectedErr string
	}{
		{
			name: "Valid Images",
			images: []ContainerImage{
				{
					Name: "hello-world:latest",
					Key:  "",
				},
				{
					Name: "rgcrprod.azurecr.us/longhornio/longhorn-ui:v1.5.1",
					Key:  "carbide-key.pub",
				},
				{
					Name: "fake-image:latest",
				},
				{
					Name: "fake-image2-with-dupe-key:latest",
					Key:  "carbide-key.pub",
				},
			},
		},
		{
			name: "No Image Name Defined",
			images: []ContainerImage{
				{
					Name: "",
					Key:  "",
				},
				{
					Key: "",
				},
			},
			expectedErr: "no image name defined",
		},
		{
			name: "Duplicate Container Image",
			images: []ContainerImage{
				{
					Name: "hello-world:latest",
					Key:  "",
				},
				{
					Name: "hello-world:latest",
					Key:  "carbide-key.pub",
				},
			},
			expectedErr: "duplicate container image found: 'hello-world:latest'",
		},
	}

	// Test
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
	// Setup
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
			expectedErr: "no chart repository URL defined",
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
			expectedErr: "no chart version defined",
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

	// Test
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

func TestValidateHaulerFiles(t *testing.T) {
	// Setup
	tests := []struct {
		name        string
		files       []File
		expectedErr string
	}{
		{
			name: "Valid Files",
			files: []File{
				{
					Name: "install.sh",
					Path: "https://get.rke2.io",
				},
			},
		},
		{
			name: "No File Name Defined",
			files: []File{
				{
					Name: "",
					Path: "https://get.rke2.io",
				},
			},
			expectedErr: "no file name defined",
		},
		{
			name: "No File Path Defined",
			files: []File{
				{
					Name: "install.sh",
					Path: "",
				},
			},
			expectedErr: "no file path defined",
		},
		{
			name: "Duplicate File",
			files: []File{
				{
					Name: "install.sh",
					Path: "https://get.rke2.io",
				},
				{
					Name: "install.sh",
					Path: "https://get.rke2.io",
				},
			},
			expectedErr: "duplicate file found: 'install.sh'",
		},
	}

	// Test
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateFiles(test.files)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
