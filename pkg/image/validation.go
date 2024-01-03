package image

import (
	"fmt"
	"strings"
)

func validate(definition *Definition) error {
	err := validateAPIVersion(definition)
	if err != nil {
		return fmt.Errorf("error validating apiVersion: %w", err)
	}
	err = validateImage(definition)
	if err != nil {
		return fmt.Errorf("error validating image: %w", err)
	}
	err = validateOperatingSystem(definition)
	if err != nil {
		return fmt.Errorf("error validating operating system: %w", err)
	}

	return nil
}

func validateAPIVersion(definition *Definition) error {
	// Open to discussion regarding how to validate the else if
	if definition.APIVersion == (Definition{}.APIVersion) {
		return fmt.Errorf("apiVersion not defined")
	} else if len(definition.APIVersion) < 2 {
		return fmt.Errorf("invalid apiVersion")
	}
	return nil
}

func validateImage(definition *Definition) error {
	if definition.Image == (Image{}) {
		return fmt.Errorf("image not defined")
	}
	if definition.Image.ImageType == (Definition{}.Image.ImageType) {
		return fmt.Errorf("imageType not defined")
	} else if definition.Image.ImageType != "iso" && definition.Image.ImageType != "raw" {
		return fmt.Errorf("invalid imageType, should be 'iso' or 'raw'")
	}
	if definition.Image.BaseImage == (Definition{}.Image.BaseImage) {
		return fmt.Errorf("baseImage not defined")
	}
	if definition.Image.OutputImageName == (Definition{}.Image.OutputImageName) {
		return fmt.Errorf("outputImageName not defined")
	}
	return nil
}

func validateOperatingSystem(definition *Definition) error {
	if checkIfOperatingSystemDefined(&definition.OperatingSystem) {
		return nil
	}
	err := validateKernalArgs(&definition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("error validating kernal args: %w", err)
	}
	err = validateSystemd(&definition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("error validating systemd args: %w", err)
	}
	err = validateUsers(&definition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("error validating users: %w", err)
	}
	return nil
}

func checkIfOperatingSystemDefined(os *OperatingSystem) bool {
	return len(os.KernelArgs) == 0 &&
		len(os.Users) == 0 &&
		len(os.Systemd.Enable) == 0 && len(os.Systemd.Disable) == 0 &&
		os.Suma.Host == "" && os.Suma.ActivationKey == "" && !os.Suma.GetSSL
}

func validateKernalArgs(os *OperatingSystem) error {
	seenKeys := make(map[string]bool)

	for _, arg := range os.KernelArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid kernel arg: %s, expected format key=value", arg)
		}

		key, value := parts[0], parts[1]
		if value == "" {
			return fmt.Errorf("kernel arg '%s' has no value", key)
		}

		if _, exists := seenKeys[key]; exists {
			return fmt.Errorf("duplicate kernel arg found: %s", key)
		}
		seenKeys[key] = true
	}

	return nil
}

func validateSystemd(os *OperatingSystem) error {
	if err := checkForDuplicates(os.Systemd.Enable, "enable"); err != nil {
		return err
	}

	if err := checkForDuplicates(os.Systemd.Disable, "disable"); err != nil {
		return err
	}

	for _, enableItem := range os.Systemd.Enable {
		for _, disableItem := range os.Systemd.Disable {
			if enableItem == disableItem {
				return fmt.Errorf("conflict found: '%s' is both enabled and disabled", enableItem)
			}
		}
	}

	return nil
}

func checkForDuplicates(items []string, listType string) error {
	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item] {
			return fmt.Errorf("duplicate found in %s: '%s'", listType, item)
		}
		seen[item] = true
	}
	return nil
}

func validateUsers(os *OperatingSystem) error {
	seenUsernames := make(map[string]bool)

	for _, user := range os.Users {
		if user.Username == "" {
			return fmt.Errorf("user entry missing username")
		}

		if user.Password == "" && user.SSHKey == "" {
			return fmt.Errorf("user '%s' must have either a password or an SSH key", user.Username)
		}

		if seenUsernames[user.Username] {
			return fmt.Errorf("duplicate username found: '%s'", user.Username)
		}

		seenUsernames[user.Username] = true
	}

	return nil
}
