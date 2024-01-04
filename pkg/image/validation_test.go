package image

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDefinition(t *testing.T) {
	// Setup
	filename := "./testdata/full-valid-example.yaml"
	configData, err := os.ReadFile(filename)
	definition, err := ParseDefinition(configData)
	require.NoError(t, err)

	// Test
	err = ValidateDefinition(definition)
	if err != nil {
		fmt.Println(err)
	}
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

func TestValidateOperatingSystemValidKernelArgs(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key1=value1", "key2=value2"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.NoError(t, err)
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

func TestValidateOperatingSystemKernelArgInvalidFormat(t *testing.T) {
	// Setup
	def := Definition{
		OperatingSystem: OperatingSystem{
			KernelArgs: []string{"key1", "key2=value2"},
		},
	}

	// Test
	err := validateKernelArgs(&def.OperatingSystem)

	// Verify
	require.ErrorContains(t, err, "invalid kernel arg")
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
					Username: "username",
					Password: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:   "ssh-rsa AAAqeCzFPRrNyA5a",
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
					Username: "",
					Password: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:   "ssh-rsa AAAqeCzFPRrNyA5a",
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
					Username: "user1",
					Password: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:   "ssh-rsa AAAqeCzFPRrNyA5a",
				},
				{
					Username: "user1",
					Password: "$6$bZfTI3Wj05fdxQcB$W",
					SSHKey:   "ssh-rsa AAAqeCzFPRrNyA5a",
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
					Username: "user1",
					Password: "",
					SSHKey:   "",
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
