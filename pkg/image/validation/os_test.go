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
					Groups: []image.OperatingSystemGroup{
						{
							Name: "eibTeam",
						},
					},
					Users: []image.OperatingSystemUser{
						{
							Username:          "danny",
							CreateHomeDir:     true,
							EncryptedPassword: "InternNoMore",
							SSHKeys:           []string{"asdf"},
						},
					},
					Suma: image.Suma{
						Host:          "example.com",
						ActivationKey: "please?",
					},
					Packages: image.Packages{
						PKGList: []string{"zsh", "git"},
						AdditionalRepos: []image.AddRepo{
							{
								URL: "myrepo.com",
							},
						},
						RegCode: "letMeIn",
					},
					IsoConfiguration: image.IsoConfiguration{
						InstallDevice: "/dev/sda",
					},
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
					Groups: []image.OperatingSystemGroup{
						{
							Name: "dupeGroup",
						},
						{
							Name: "dupeGroup",
						},
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
					IsoConfiguration: image.IsoConfiguration{
						InstallDevice: "/dev/sda",
					},
					RawConfiguration: image.RawConfiguration{
						DiskSize: "64",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"Kernel arguments must be specified as 'key=value'.",
				"Systemd conflict found, 'confusedUser' is both enabled and disabled.",
				"Duplicate group name found: dupeGroup",
				"User 'danny' must have either a password or at least one SSH key.",
				"The 'host' field is required for the 'suma' section.",
				fmt.Sprintf("The 'isoConfiguration/installDevice' field can only be used when 'imageType' is '%s'.", image.TypeISO),
				"The 'rawConfiguration/diskSize' field must be an integer followed by a suffix of either 'M', 'G', or 'T'.",
				"You cannot simultaneously configure rawConfiguration and isoConfiguration, regardless of image type.",
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
			}

			for _, expectedMessage := range test.ExpectedFailedMessages {
				assert.Contains(t, foundMessages, expectedMessage)
			}
		})
	}
}

func TestValidateGroups(t *testing.T) {
	tests := map[string]struct {
		Groups                 []image.OperatingSystemGroup
		ExpectedFailedMessages []string
	}{
		`no groups`: {
			Groups: []image.OperatingSystemGroup{},
		},
		`valid groups`: {
			Groups: []image.OperatingSystemGroup{
				{
					Name: "group1",
				},
				{
					Name: "group2",
				},
			},
		},
		`missing group name`: {
			Groups: []image.OperatingSystemGroup{
				{},
			},
			ExpectedFailedMessages: []string{
				"The 'name' field is required for all entries under 'groups'.",
			},
		},
		`duplicate group name`: {
			Groups: []image.OperatingSystemGroup{
				{
					Name: "group1",
				},
				{
					Name: "group1",
				},
				{
					Name: "group2",
				},
				{
					Name: "group2",
				},
				{
					Name: "group3",
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate group name found: group1",
				"Duplicate group name found: group2",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os := image.OperatingSystem{
				Groups: test.Groups,
			}
			failures := validateGroups(&os)
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
					CreateHomeDir:     true,
					EncryptedPassword: "foo",
					SSHKeys:           []string{"key"},
				},
				{
					Username:          "rhys",
					EncryptedPassword: "pm-4-life",
				},
				{
					Username:      "atanas",
					CreateHomeDir: true,
					SSHKeys:       []string{"key2"},
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
				"User 'danny' must have either a password or at least one SSH key.",
			},
		},
		`duplicate user`: {
			Users: []image.OperatingSystemUser{
				{
					Username:          "ivo",
					EncryptedPassword: "password1",
				},
				{
					Username:      "ivo",
					CreateHomeDir: true,
					SSHKeys:       []string{"key1"},
				},
			},
			ExpectedFailedMessages: []string{
				"Duplicate username found: ivo",
			},
		},
		`ssh key and no create home`: {
			Users: []image.OperatingSystemUser{
				{
					Username: "edu",
					SSHKeys:  []string{"key1"},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'createHomeDir' attribute must be set to 'true' if at least one SSH key is specified.",
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
				PKGList: []string{"foo"},
				AdditionalRepos: []image.AddRepo{
					{
						URL: "myrepo",
					},
				},
				RegCode: "regcode",
			},
		},
		`empty package`: {
			Packages: image.Packages{
				PKGList: []string{"foo", "bar", ""},
			},
			ExpectedFailedMessages: []string{
				"The 'packageList' field cannot contain empty values.",
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
				AdditionalRepos: []image.AddRepo{
					{
						URL: "foo",
					},
					{
						URL: "bar",
					},
					{
						URL: "foo",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'additionalRepos' field contains duplicate repos: foo",
			},
		},
		`missing repo url`: {
			Packages: image.Packages{
				AdditionalRepos: []image.AddRepo{
					{
						URL: "",
					},
					{
						URL: "foo",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'url' field is required for all entries under 'additionalRepos'.",
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
		`iso install device specified`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeISO,
				},
				OperatingSystem: image.OperatingSystem{
					IsoConfiguration: image.IsoConfiguration{
						InstallDevice: "/dev/sda",
					},
				},
			},
		},
		`not iso install device`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					IsoConfiguration: image.IsoConfiguration{
						InstallDevice: "/dev/sda",
					},
				},
			},
			ExpectedFailedMessages: []string{
				fmt.Sprintf("The 'isoConfiguration/installDevice' field can only be used when 'imageType' is '%s'.", image.TypeISO),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			def := test.Definition
			failures := validateIsoConfig(&def)
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

func TestValidateRawConfiguration(t *testing.T) {
	tests := map[string]struct {
		Definition             image.Definition
		ExpectedFailedMessages []string
	}{
		`not included`: {
			Definition: image.Definition{},
		},
		`diskSize specified and valid`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						DiskSize: "64G",
					},
				},
			},
		},
		`diskSize invalid as invalid suffix`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						DiskSize: "130B",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'rawConfiguration/diskSize' field must be an integer followed by a suffix of either 'M', 'G', or 'T'.",
			},
		},
		`diskSize invalid as zero`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						DiskSize: "0G",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'rawConfiguration/diskSize' field must be an integer followed by a suffix of either 'M', 'G', or 'T'.",
			},
		},
		`diskSize invalid as lowercase character`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						DiskSize: "100g",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'rawConfiguration/diskSize' field must be an integer followed by a suffix of either 'M', 'G', or 'T'.",
			},
		},
		`diskSize invalid as negative number`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						DiskSize: "-100G",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'rawConfiguration/diskSize' field must be an integer followed by a suffix of either 'M', 'G', or 'T'.",
			},
		},
		`diskSize invalid as no number provided`: {
			Definition: image.Definition{
				Image: image.Image{
					ImageType: image.TypeRAW,
				},
				OperatingSystem: image.OperatingSystem{
					RawConfiguration: image.RawConfiguration{
						DiskSize: "G",
					},
				},
			},
			ExpectedFailedMessages: []string{
				"The 'rawConfiguration/diskSize' field must be an integer followed by a suffix of either 'M', 'G', or 'T'.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			def := test.Definition
			failures := validateRawConfig(&def)
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

func TestValidateTimeSync(t *testing.T) {
	tests := map[string]struct {
		Time                   image.Time
		ExpectedFailedMessages []string
	}{
		`not included`: {
			Time: image.Time{},
		},
		`forceWait specified and only NTP pools configured`: {
			Time: image.Time{
				Timezone: "Europe/London",
				NtpConfiguration: image.NtpConfiguration{
					Pools:     []string{"2.suse.pool.ntp.org"},
					ForceWait: true,
				},
			},
		},
		`forceWait specified and only NTP servers configured`: {
			Time: image.Time{
				Timezone: "Europe/London",
				NtpConfiguration: image.NtpConfiguration{
					Servers:   []string{"10.0.0.1", "10.0.0.2"},
					ForceWait: true,
				},
			},
		},
		`forceWait specified and NTP sources missing`: {
			Time: image.Time{
				Timezone: "Europe/London",
				NtpConfiguration: image.NtpConfiguration{
					ForceWait: true,
				},
			},
			ExpectedFailedMessages: []string{
				"If you're wanting to wait for NTP synchronization at boot, please ensure that you provide at least one NTP time source.",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			os := image.OperatingSystem{
				Time: test.Time,
			}
			failures := validateTimeSync(&os)
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
