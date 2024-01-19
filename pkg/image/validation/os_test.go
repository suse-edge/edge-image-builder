package validation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestValidateOperatingSystem(t *testing.T) {
	tests := map[string]struct {
		Definition             image.Definition
		ExpectedFailedMessages []string
	}{
		`no os defined`: {
			Definition: image.Definition{},
		},
		`all valid`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeISO,
				},
				OperatingSystem: image.OperatingSystem{
					KernelArgs: []string{"foo=bar", "baz"},
					Systemd: image.Systemd{
						Enable:  []string{"runMe"},
						Disable: []string{"dontRunMe"},
					},
					Users: []image.OperatingSystemUser{
						{
							Username:          "danny",
							EncryptedPassword: "InternNoMore",
							SSHKey:            "asdf",
						},
					},
					Suma: image.Suma{
						Host:          "example.com",
						ActivationKey: "please?",
					},
					Packages: image.Packages{
						PKGList:         []string{"zsh", "git"},
						AdditionalRepos: []string{"myrepo.com"},
						RegCode:         "letMeIn",
					},
					Unattended:    true,
					InstallDevice: "/dev/sda",
				},
			},
		},
		`all invalid`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					KernelArgs: []string{"foo="},
					Systemd: image.Systemd{
						Enable:  []string{"confusedUser"},
						Disable: []string{"confusedUser"},
					},
					Users: []image.OperatingSystemUser{
						{
							Username: "danny",
						},
					},
					Suma: image.Suma{
						ActivationKey: "please?",
					},
					Packages: image.Packages{
						PKGList: []string{"zsh", "git"},
					},
					Unattended:    true,
					InstallDevice: "/dev/sda",
				},
			},
			ExpectedFailedMessages: []string{
				"Kernel arguments must be specified as 'key=value'.",
				"Systemd conflict found, 'confusedUser' is both enabled and disabled.",
				"User 'danny' must have either a password or SSH key.",
				"The 'host' field is required for the 'suma' section.",
				"When including the 'packageList' field, either additional repositories or a registration code must be included.",
				fmt.Sprintf("The 'unattended' field can only be used when 'imageType' is '%s'.", image.TypeISO),
				fmt.Sprintf("The 'installDevice' field can only be used when 'imageType' is '%s'.", image.TypeISO),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			def := test.Definition
			ctx := image.Context{
				ImageDefinition: &def,
			}
			failures := validateOperatingSystem(&ctx)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))
		})
	}
}

func TestIsOperatingSystemDefined(t *testing.T) {
	tests := map[string]struct {
		OS       image.OperatingSystem
		Expected bool
	}{
		`empty operating system`: {
			OS:       image.OperatingSystem{},
			Expected: false,
		},
		`with kernel args`: {
			OS: image.OperatingSystem{
				KernelArgs: []string{"foo=bar"},
			},
			Expected: true,
		},
		`with users`: {
			OS: image.OperatingSystem{
				Users: []image.OperatingSystemUser{
					{Username: "jdob"},
				},
			},
			Expected: true,
		},
		`with systemd enable list`: {
			OS: image.OperatingSystem{
				Systemd: image.Systemd{
					Enable: []string{"foo"},
				},
			},
			Expected: true,
		},
		`with systemd disable list`: {
			OS: image.OperatingSystem{
				Systemd: image.Systemd{
					Disable: []string{"bar"},
				},
			},
			Expected: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os := test.OS
			result := isOperatingSystemDefined(&os)
			assert.Equal(t, test.Expected, result)
		})
	}
}

func TestValidateKernelArgs(t *testing.T) {
	tests := map[string]struct {
		OS                     image.OperatingSystem
		ExpectedFailedMessages []string
	}{
		`valid test`: {
			OS: image.OperatingSystem{
				KernelArgs: []string{"foo=bar", "baz"},
			},
		},
		`no key`: {
			OS: image.OperatingSystem{
				KernelArgs: []string{"foo="},
			},
			ExpectedFailedMessages: []string{
				"Kernel arguments must be specified as 'key=value'.",
			},
		},
		`no value`: {
			OS: image.OperatingSystem{
				KernelArgs: []string{"=bar"},
			},
			ExpectedFailedMessages: []string{
				"Kernel arguments must be specified as 'key=value'.",
			},
		},
		`duplicate key`: {
			OS: image.OperatingSystem{
				KernelArgs: []string{"foo=bar", "foo=wombat"},
			},
			ExpectedFailedMessages: []string{
				"Duplicate kernel argument found: foo",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os := test.OS
			failures := validateKernelArgs(&os)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
				assert.Equal(t, osComponent, foundValidation.Component)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestValidateSystemd(t *testing.T) {
	tests := map[string]struct {
		Systemd                image.Systemd
		ExpectedFailedMessages []string
	}{
		`no systemd`: {
			Systemd: image.Systemd{},
		},
		`valid enable and disable`: {
			Systemd: image.Systemd{
				Enable:  []string{"foo", "bar"},
				Disable: []string{"baz"},
			},
		},
		`enable and disable duplicates`: {
			Systemd: image.Systemd{
				Enable:  []string{"foo", "foo", "baz", "baz"},
				Disable: []string{"bar", "bar"},
			},
			ExpectedFailedMessages: []string{
				"Systemd enable list contains duplicate entries: foo, baz",
				"Systemd disable list contains duplicate entries: bar",
			},
		},
		`conflict`: {
			Systemd: image.Systemd{
				Enable:  []string{"foo", "bar", "zombie"},
				Disable: []string{"foo", "bar", "wombat"},
			},
			ExpectedFailedMessages: []string{
				"Systemd conflict found, 'foo' is both enabled and disabled.",
				"Systemd conflict found, 'bar' is both enabled and disabled.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os := image.OperatingSystem{
				Systemd: test.Systemd,
			}
			failures := validateSystemd(&os)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
				assert.Equal(t, osComponent, foundValidation.Component)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestValidateUsers(t *testing.T) {
	tests := map[string]struct {
		Users                  []image.OperatingSystemUser
		ExpectedFailedMessages []string
	}{
		`no users`: {
			Users: []image.OperatingSystemUser{},
		},
		`valid users`: {
			Users: []image.OperatingSystemUser{
				{
					Username:          "jay",
					EncryptedPassword: "foo",
					SSHKey:            "key",
				},
				{
					Username:          "rhys",
					EncryptedPassword: "pm-4-life",
				},
				{
					Username: "atanas",
					SSHKey:   "key2",
				},
			},
		},
		`user no credentials`: {
			Users: []image.OperatingSystemUser{
				{
					Username: "danny",
				},
			},
			ExpectedFailedMessages: []string{
				"User 'danny' must have either a password or SSH key.",
			},
		},
		`duplicate user`: {
			Users: []image.OperatingSystemUser{
				{
					Username:          "ivo",
					EncryptedPassword: "password1",
				},
				{
					Username: "ivo",
					SSHKey:   "key1",
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate username found: ivo",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os := image.OperatingSystem{
				Users: test.Users,
			}
			failures := validateUsers(&os)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
				assert.Equal(t, osComponent, foundValidation.Component)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestValidateSuma(t *testing.T) {
	tests := map[string]struct {
		Suma                   image.Suma
		ExpectedFailedMessages []string
	}{
		`no suma`: {
			Suma: image.Suma{},
		},
		`valid suma`: {
			Suma: image.Suma{
				Host:          "non-http",
				ActivationKey: "foo",
			},
		},
		`no host`: {
			Suma: image.Suma{
				ActivationKey: "foo",
			},
			ExpectedFailedMessages: []string{
				"The 'host' field is required for the 'suma' section.",
			},
		},
		`http host`: {
			Suma: image.Suma{
				Host:          "http://example.com",
				ActivationKey: "foo",
			},
			ExpectedFailedMessages: []string{
				"The suma 'host' field may not contain 'http://' or 'https://'",
			},
		},
		`no activation key`: {
			Suma: image.Suma{
				Host: "valid",
			},
			ExpectedFailedMessages: []string{
				"The 'activationKey' field is required for the 'suma' section.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os := image.OperatingSystem{
				Suma: test.Suma,
			}
			failures := validateSuma(&os)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
				assert.Equal(t, osComponent, foundValidation.Component)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestPackages(t *testing.T) {
	tests := map[string]struct {
		Packages               image.Packages
		ExpectedFailedMessages []string
	}{
		`no packages`: {
			Packages: image.Packages{},
		},
		`valid`: {
			Packages: image.Packages{
				PKGList:         []string{"foo"},
				AdditionalRepos: []string{"myrepo"},
				RegCode:         "regcode",
			},
		},
		`package list only`: {
			Packages: image.Packages{
				PKGList: []string{"foo", "bar"},
			},
			ExpectedFailedMessages: []string{
				"When including the 'packageList' field, either additional repositories or a registration code must be included.",
			},
		},
		`duplicate packages`: {
			Packages: image.Packages{
				PKGList: []string{"foo", "bar", "foo", "bar", "baz"},
				RegCode: "regcode",
			},
			ExpectedFailedMessages: []string{
				"The 'packageList' field contains duplicate packages: foo, bar",
			},
		},
		`duplicate repos`: {
			Packages: image.Packages{
				AdditionalRepos: []string{"foo", "bar", "foo"},
			},
			ExpectedFailedMessages: []string{
				"The 'additionalRepos' field contains duplicate repos: foo",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os := image.OperatingSystem{
				Packages: test.Packages,
			}
			failures := validatePackages(&os)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
				assert.Equal(t, osComponent, foundValidation.Component)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestValidateUnattended(t *testing.T) {
	tests := map[string]struct {
		Definition             image.Definition
		ExpectedFailedMessages []string
	}{
		`not included`: {
			Definition: image.Definition{},
		},
		`iso both specified`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeISO,
				},
				OperatingSystem: image.OperatingSystem{
					Unattended:    true,
					InstallDevice: "/dev/sda",
				},
			},
		},
		`not iso unattended`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					Unattended: true,
				},
			},
			ExpectedFailedMessages: []string{
				fmt.Sprintf("The 'unattended' field can only be used when 'imageType' is '%s'.", image.TypeISO),
			},
		},
		`not iso install device`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					InstallDevice: "/dev/sda",
				},
			},
			ExpectedFailedMessages: []string{
				fmt.Sprintf("The 'installDevice' field can only be used when 'imageType' is '%s'.", image.TypeISO),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			def := test.Definition
			failures := validateUnattended(&def)
			assert.Len(t, failures, len(test.ExpectedFailedMessages))

			var foundMessages []string
			for _, foundValidation := range failures {
				foundMessages = append(foundMessages, foundValidation.UserMessage)
				assert.Equal(t, osComponent, foundValidation.Component)
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}
