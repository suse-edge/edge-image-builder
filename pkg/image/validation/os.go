package validation

import (
	"fmt"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	osComponent = "Operating System"
)

func validateOperatingSystem(ctx *image.Context) []FailedValidation {
	def := ctx.ImageDefinition

	var failures []FailedValidation

	if !isOperatingSystemDefined(&def.OperatingSystem) {
		return failures
	}

	failures = append(failures, validateKernelArgs(&def.OperatingSystem)...)
	failures = append(failures, validateSystemd(&def.OperatingSystem)...)
	failures = append(failures, validateUsers(&def.OperatingSystem)...)
	failures = append(failures, validateSuma(&def.OperatingSystem)...)
	failures = append(failures, validatePackages(&def.OperatingSystem)...)
	failures = append(failures, validateUnattended(def)...)

	return failures
}

func isOperatingSystemDefined(os *image.OperatingSystem) bool {
	return !(len(os.KernelArgs) == 0 &&
		len(os.Users) == 0 &&
		len(os.Systemd.Enable) == 0 &&
		len(os.Systemd.Disable) == 0 &&
		os.Suma == (image.Suma{}))
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
					Component:   osComponent,
				})
			}
		}

		if _, exists := seenKeys[key]; exists {
			failures = append(failures, FailedValidation{
				UserMessage: fmt.Sprintf("Duplicate kernel argument found: %s", key),
				Component:   osComponent,
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
			Component:   osComponent,
		})
	}

	if duplicates := findDuplicates(os.Systemd.Disable); len(duplicates) > 0 {
		duplicateValues := strings.Join(duplicates, ", ")
		msg := fmt.Sprintf("Systemd disable list contains duplicate entries: %s", duplicateValues)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
			Component:   osComponent,
		})
	}

	for _, enableItem := range os.Systemd.Enable {
		for _, disableItem := range os.Systemd.Disable {
			if enableItem == disableItem {
				msg := fmt.Sprintf("Systemd conflict found, '%s' is both enabled and disabled.", enableItem)
				failures = append(failures, FailedValidation{
					UserMessage: msg,
					Component:   osComponent,
				})
			}
		}
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
				Component:   osComponent,
			})
		}

		if user.EncryptedPassword == "" && user.SSHKey == "" {
			msg := fmt.Sprintf("User '%s' must have either a password or SSH key.", user.Username)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
				Component:   osComponent,
			})
		}

		if seenUsernames[user.Username] {
			msg := fmt.Sprintf("Duplicate username found: %s", user.Username)
			failures = append(failures, FailedValidation{
				UserMessage: msg,
				Component:   osComponent,
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
			Component:   osComponent,
		})
	}
	if strings.HasPrefix(os.Suma.Host, "http") {
		failures = append(failures, FailedValidation{
			UserMessage: "The suma 'host' field may not contain 'http://' or 'https://'",
			Component:   osComponent,
		})
	}
	if os.Suma.ActivationKey == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "The 'activationKey' field is required for the 'suma' section.",
			Component:   osComponent,
		})
	}

	return failures
}

func validatePackages(os *image.OperatingSystem) []FailedValidation {
	var failures []FailedValidation

	if duplicates := findDuplicates(os.Packages.PKGList); len(duplicates) > 0 {
		duplicateValues := strings.Join(duplicates, ", ")
		msg := fmt.Sprintf("The 'packageList' field contains duplicate packages: %s", duplicateValues)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
			Component:   osComponent,
		})
	}

	if duplicates := findDuplicates(os.Packages.AdditionalRepos); len(duplicates) > 0 {
		duplicateValues := strings.Join(duplicates, ", ")
		msg := fmt.Sprintf("The 'additionalRepos' field contains duplicate repos: %s", duplicateValues)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
			Component:   osComponent,
		})
	}

	if len(os.Packages.PKGList) > 0 && len(os.Packages.AdditionalRepos) == 0 && os.Packages.RegCode == "" {
		failures = append(failures, FailedValidation{
			UserMessage: "When including the 'packageList' field, either additional repositories or a registration code must be included.",
			Component:   osComponent,
		})
	}

	return failures
}

func validateUnattended(def *image.Definition) []FailedValidation {
	var failures []FailedValidation

	if def.Image.ImageType != image.TypeISO && def.OperatingSystem.Unattended {
		msg := fmt.Sprintf("The 'unattended' field can only be used when 'imageType' is '%s'.", image.TypeISO)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
			Component:   osComponent,
		})
	}

	if def.Image.ImageType != image.TypeISO && def.OperatingSystem.InstallDevice != "" {
		msg := fmt.Sprintf("The 'installDevice' field can only be used when 'imageType' is '%s'.", image.TypeISO)
		failures = append(failures, FailedValidation{
			UserMessage: msg,
			Component:   osComponent,
		})
	}

	return failures
}
