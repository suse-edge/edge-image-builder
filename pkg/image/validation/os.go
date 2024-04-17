package validation

import (
	"fmt"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	osComponent = "Operating System"
)

func validateOperatingSystem(ctx *image.Context) []FailedValidation {
	def := ctx.ImageDefinition

	var failures []FailedValidation

	failures = append(failures, validateKernelArgs(&def.OperatingSystem)...)
	failures = append(failures, validateSystemd(&def.OperatingSystem)...)
	failures = append(failures, validateGroups(&def.OperatingSystem)...)
	failures = append(failures, validateUsers(&def.OperatingSystem)...)
	failures = append(failures, validateSuma(&def.OperatingSystem)...)
	failures = append(failures, validatePackages(&def.OperatingSystem)...)
	failures = append(failures, validateTimeSync(&def.OperatingSystem)...)
	failures = append(failures, validateIsoConfig(def)...)
	failures = append(failures, validateRawConfig(def)...)

	return failures
}

func validateKernelArgs(os *image.OperatingSystem) []FailedValidation {
	var failures []FailedValidation

	seenKeys := make(map[string]bool)
	for _, arg := range os.KernelArgs {
		key := arg

		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			var value string
			key, value = parts[0], parts[1]
			if key == "" || value == "" {
				failures = append(failures, FailedValidation{
					UserMessage: "Kernel arguments must be specified as 'key=value'.",
				})
			}
		}

		if _, exists := seenKeys[key]; exists {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Duplicate kernel argument found: %s", key),
			})
		}
		seenKeys[key] = true
	}

	return failures
}

func validateSystemd(os *image.OperatingSystem) []FailedValidation {
	var failures []FailedValidation

	if duplicates := findDuplicates(os.Systemd.Enable); len(duplicates) > 0 {
		duplicateValues := strings.Join(duplicates, ", ")
		msg := fmt.Sprintf("Systemd enable list contains duplicate entries: %s", duplicateValues)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	if duplicates := findDuplicates(os.Systemd.Disable); len(duplicates) > 0 {
		duplicateValues := strings.Join(duplicates, ", ")
		msg := fmt.Sprintf("Systemd disable list contains duplicate entries: %s", duplicateValues)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	for _, enableItem := range os.Systemd.Enable {
		for _, disableItem := range os.Systemd.Disable {
			if enableItem == disableItem {
				msg := fmt.Sprintf("Systemd conflict found, '%s' is both enabled and disabled.", enableItem)
				failures = append(failures, FailedValidation{
					UserMessage: msg,
				})
			}
		}
	}

	return failures
}

func validateGroups(os *image.OperatingSystem) []FailedValidation {
	var failures []FailedValidation

	// The script is idempotent and will not fail on creating a duplicate group,
	// but for consistency validate that duplicates aren't in the definition.
	seenGroupNames := make(map[string]bool)
	for _, group := range os.Groups {
		if group.Name == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'name' field is required for all entries under 'groups'.",
			})
		}

		if seenGroupNames[group.Name] {
			msg := fmt.Sprintf("Duplicate group name found: %s", group.Name)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}
		seenGroupNames[group.Name] = true
	}

	return failures
}

func validateUsers(os *image.OperatingSystem) []FailedValidation {
	var failures []FailedValidation

	seenUsernames := make(map[string]bool)
	for _, user := range os.Users {
		if user.Username == "" {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'username' field is required for all entries under 'users'.",
			})
		}

		if user.EncryptedPassword == "" && len(user.SSHKeys) == 0 {
			msg := fmt.Sprintf("User '%s' must have either a password or at least one SSH key.", user.Username)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}

		if !user.CreateHomeDir && len(user.SSHKeys) > 0 {
			failures = append(failures, FailedValidation{
				UserMessage: "The 'createHomeDir' attribute must be set to 'true' if at least one SSH key is specified.",
			})
		}

		if seenUsernames[user.Username] {
			msg := fmt.Sprintf("Duplicate username found: %s", user.Username)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}
		seenUsernames[user.Username] = true
	}

	return failures
}

func validateSuma(os *image.OperatingSystem) []FailedValidation {
	var failures []FailedValidation

	if os.Suma == (image.Suma{}) {
		return failures
	}
	if os.Suma.Host == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'host' field is required for the 'suma' section.",
		})
	}
	if strings.HasPrefix(os.Suma.Host, "http") {
		failures = append(failures, FailedValidation{
			UserMessage: "The suma 'host' field may not contain 'http://' or 'https://'",
		})
	}
	if os.Suma.ActivationKey == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'activationKey' field is required for the 'suma' section.",
		})
	}

	return failures
}

func validatePackages(os *image.OperatingSystem) []FailedValidation {
	var failures []FailedValidation

	if slices.Contains(os.Packages.PKGList, "") {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'packageList' field cannot contain empty values.",
		})
	}

	if duplicates := findDuplicates(os.Packages.PKGList); len(duplicates) > 0 {
		duplicateValues := strings.Join(duplicates, ", ")
		msg := fmt.Sprintf("The 'packageList' field contains duplicate packages: %s", duplicateValues)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	// It is possible to only provide `additionalRepos` without listing any packages
	// under `packageList` in the cases where RPMs are side-loaded under the `/rpms` directory.
	if len(os.Packages.AdditionalRepos) > 0 {
		var repoURLs []string

		for _, repo := range os.Packages.AdditionalRepos {
			if repo.URL == "" {
				msg := "The 'url' field is required for all entries under 'additionalRepos'."
				failures = append(failures, FailedValidation{
					UserMessage: msg,
				})
			}

			repoURLs = append(repoURLs, repo.URL)
		}

		if duplicates := findDuplicates(repoURLs); len(duplicates) > 0 {
			duplicateValues := strings.Join(duplicates, ", ")
			msg := fmt.Sprintf("The 'additionalRepos' field contains duplicate repos: %s", duplicateValues)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
			})
		}
	}

	return failures
}

func validateIsoConfig(def *image.Definition) []FailedValidation {
	var failures []FailedValidation

	if def.Image.ImageType != image.TypeISO && def.OperatingSystem.IsoConfiguration.InstallDevice != "" {
		msg := fmt.Sprintf("The 'isoConfiguration/installDevice' field can only be used when 'imageType' is '%s'.", image.TypeISO)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	return failures
}

func validateRawConfig(def *image.Definition) []FailedValidation {
	var failures []FailedValidation

	if def.OperatingSystem.RawConfiguration.DiskSize == "" {
		return nil
	}

	if def.Image.ImageType != image.TypeRAW {
		msg := fmt.Sprintf("The 'rawConfiguration/diskSize' field can only be used when 'imageType' is '%s'.", image.TypeRAW)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	if def.OperatingSystem.IsoConfiguration.InstallDevice != "" {
		msg := "You cannot simultaneously configure rawConfiguration and isoConfiguration, regardless of image type."
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	if !def.OperatingSystem.RawConfiguration.DiskSize.IsValid() {
		msg := "The 'rawConfiguration/diskSize' field must be an integer followed by a suffix of either 'M', 'G', or 'T'."
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	return failures
}

func validateTimeSync(os *image.OperatingSystem) []FailedValidation {
	var failures []FailedValidation

	if !os.Time.NtpConfiguration.ForceWait {
		return nil
	}

	if len(os.Time.NtpConfiguration.Pools) == 0 && len(os.Time.NtpConfiguration.Servers) == 0 {
		msg := "If you're wanting to wait for NTP synchronization at boot, please ensure that you provide at least one NTP time source."
		failures = append(failures, FailedValidation{
			UserMessage: msg,
		})
	}

	return failures
}
