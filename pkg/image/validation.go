package image

import (
	"fmt"
	"strings"
)

func ValidateDefinition(definition *Definition) error {
	err := validateImage(definition)
	if err != nil {
		return fmt.Errorf("error validating image: %w", err)
	}
	err = validateOperatingSystem(definition)
	if err != nil {
		return fmt.Errorf("error validating operating system: %w", err)
	}
	err = validateHauler(definition)
	if err != nil {
		return fmt.Errorf("error validating hauler: %w", err)
	}

	return nil
}

func validateImage(definition *Definition) error {
	if definition.Image == (Image{}) {
		return fmt.Errorf("image not defined")
	}
	if definition.Image.ImageType == "" {
		return fmt.Errorf("imageType not defined")
	} else if definition.Image.ImageType != "iso" && definition.Image.ImageType != "raw" {
		return fmt.Errorf("invalid imageType, should be 'iso' or 'raw'")
	}
	if definition.Image.BaseImage == "" {
		return fmt.Errorf("baseImage not defined")
	}
	if definition.Image.OutputImageName == "" {
		return fmt.Errorf("outputImageName not defined")
	}

	return nil
}

func validateOperatingSystem(definition *Definition) error {
	if checkIfOperatingSystemDefined(&definition.OperatingSystem) {
		return nil
	}
	err := validateKernelArgs(&definition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("error validating kernel args: %w", err)
	}
	err = validateSystemd(&definition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("error validating systemd args: %w", err)
	}
	err = validateUsers(&definition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("error validating users: %w", err)
	}
	err = validateSuma(&definition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("error validating suma: %w", err)
	}

	return nil
}

func checkIfOperatingSystemDefined(os *OperatingSystem) bool {
	return len(os.KernelArgs) == 0 &&
		len(os.Users) == 0 &&
		len(os.Systemd.Enable) == 0 && len(os.Systemd.Disable) == 0 &&
		os.Suma == (Suma{})
}

func validateKernelArgs(os *OperatingSystem) error {
	seenKeys := make(map[string]bool)

	for _, arg := range os.KernelArgs {
		key := arg

		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			var value string
			key, value = parts[0], parts[1]
			if key == "" {
				return fmt.Errorf("kernel arg value '%s' has no key", value)
			}
			if value == "" {
				return fmt.Errorf("kernel arg '%s' has no value", key)
			}
		}

		if _, exists := seenKeys[key]; exists {
			return fmt.Errorf("duplicate kernel arg found: '%s'", key)
		}
		seenKeys[key] = true
	}

	return nil
}

func validateSystemd(os *OperatingSystem) error {
	if duplicate := checkForDuplicates(os.Systemd.Enable); duplicate != "" {
		return fmt.Errorf("enable list contains duplicate: %s", duplicate)
	}

	if duplicate := checkForDuplicates(os.Systemd.Disable); duplicate != "" {
		return fmt.Errorf("disable list contains duplicate: %s", duplicate)
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

func checkForDuplicates(items []string) string {
	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item] {
			return item
		}
		seen[item] = true
	}

	return ""
}

func validateUsers(os *OperatingSystem) error {
	seenUsernames := make(map[string]bool)

	for _, user := range os.Users {
		if user.Username == "" {
			return fmt.Errorf("user entry missing username")
		}

		if user.EncryptedPassword == "" && user.SSHKey == "" {
			return fmt.Errorf("user '%s' must have either a password or an SSH key", user.Username)
		}

		if seenUsernames[user.Username] {
			return fmt.Errorf("duplicate username found: '%s'", user.Username)
		}
		seenUsernames[user.Username] = true
	}

	return nil
}

func validateSuma(os *OperatingSystem) error {
	if os.Suma == (Suma{}) {
		return nil
	}
	if os.Suma.Host == "" {
		return fmt.Errorf("no host defined")
	}
	if strings.HasPrefix(os.Suma.Host, "http") {
		return fmt.Errorf("invalid hostname, hostname should not contain 'http://' or 'https://'")
	}
	if os.Suma.ActivationKey == "" {
		return fmt.Errorf("no activation key defined")
	}

	return nil
}

func validateHauler(definition *Definition) error {
	if checkIfHaulerDefined(definition) {
		return nil
	}
	err := validateContainerImages(definition.EmbeddedArtifactRegistry.ContainerImages)
	if err != nil {
		return fmt.Errorf("error validating container images: %w", err)
	}
	err = validateHelmCharts(definition.EmbeddedArtifactRegistry.HelmCharts)
	if err != nil {
		return fmt.Errorf("error validating helm charts: %w", err)
	}

	return nil
}

func checkIfHaulerDefined(definition *Definition) bool {
	return len(definition.EmbeddedArtifactRegistry.HelmCharts) == 0 && len(definition.EmbeddedArtifactRegistry.ContainerImages) == 0
}

func validateContainerImages(containerImages []ContainerImage) error {
	seenContainerImages := make(map[string]bool)

	for _, image := range containerImages {
		if image.Name == "" {
			return fmt.Errorf("no image name defined")
		}

		if seenContainerImages[image.Name] {
			return fmt.Errorf("duplicate container image found: '%s'", image.Name)
		}
		seenContainerImages[image.Name] = true
	}

	return nil
}

func validateHelmCharts(charts []HelmChart) error {
	seenCharts := make(map[string]bool)

	for _, chart := range charts {
		if chart.Name == "" {
			return fmt.Errorf("no chart name defined")
		}
		if chart.RepoURL == "" {
			return fmt.Errorf("no chart repository URL defined for '%s'", chart.Name)
		}
		if chart.Version == "" {
			return fmt.Errorf("no chart version defined for '%s'", chart.Name)
		}
		if !strings.HasPrefix(chart.RepoURL, "http") {
			return fmt.Errorf("invalid chart respository url, does not start with 'http://' or 'https://'")
		}

		if seenCharts[chart.Name] {
			return fmt.Errorf("duplicate chart found: '%s'", chart.Name)
		}
		seenCharts[chart.Name] = true
	}

	return nil
}
