package image

import (
	"fmt"
	"slices"
	"strings"
)

func ValidateDefinition(definition *Definition) error {
	if err := validateImage(definition); err != nil {
		return fmt.Errorf("error validating image: %w", err)
	}

	if err := validateOperatingSystem(definition); err != nil {
		return fmt.Errorf("error validating operating system: %w", err)
	}

	if err := validateEmbeddedArtifactRegistry(definition); err != nil {
		return fmt.Errorf("error validating embedded artifact registry: %w", err)
	}

	if err := validateKubernetes(definition); err != nil {
		return fmt.Errorf("error validating kubernetes: %w", err)
	}

	return nil
}

func validateImage(definition *Definition) error {
	if definition.Image == (Image{}) {
		return fmt.Errorf("image not defined")
	}
	if definition.Image.ImageType == "" {
		return fmt.Errorf("imageType not defined")
	} else if definition.Image.ImageType != TypeISO && definition.Image.ImageType != TypeRAW {
		return fmt.Errorf("imageType must be '%s' or '%s'", TypeISO, TypeRAW)
	}
	if definition.Image.Arch == "" {
		return fmt.Errorf("arch not defined")
	} else if definition.Image.Arch != ArchTypeX86 && definition.Image.Arch != ArchTypeARM {
		return fmt.Errorf("arch must be '%s' or '%s'", ArchTypeX86, ArchTypeARM)
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
	err = validatePackages(&definition.OperatingSystem)
	if err != nil {
		return fmt.Errorf("error validating packages: %w", err)
	}
	err = validateUnattended(definition)
	if err != nil {
		return fmt.Errorf("error validating unattended mode: %w", err)
	}

	return nil
}

func validateKubernetes(definition *Definition) error {
	if definition.Kubernetes.Version == "" {
		// Not configured
		return nil
	}

	// TODO: Validate config file(s?)

	if definition.Kubernetes.Network.APIVIP == "" {
		return fmt.Errorf("virtual API address is not provided")
	}

	if definition.Kubernetes.Network.APIHost == "" {
		return fmt.Errorf("API host is not provided")
	}

	switch len(definition.Kubernetes.Nodes) {
	case 0:
		return fmt.Errorf("node list is empty")
	case 1:
		node := definition.Kubernetes.Nodes[0]

		if node.Type != KubernetesNodeTypeServer {
			return fmt.Errorf("node type in single node cluster must be 'server'")
		}

		if node.Hostname == "" {
			return fmt.Errorf("node hostname cannot be empty")
		}
	default:
		var nodeTypes []string
		var nodeNames []string

		for _, node := range definition.Kubernetes.Nodes {
			if node.Hostname == "" {
				return fmt.Errorf("node hostname cannot be empty")
			}

			if node.Type != KubernetesNodeTypeServer && node.Type != KubernetesNodeTypeAgent {
				return fmt.Errorf("invalid node type: %s", node.Type)
			}

			nodeNames = append(nodeNames, node.Hostname)
			nodeTypes = append(nodeTypes, node.Type)
		}

		if duplicate := checkForDuplicates(nodeNames); duplicate != "" {
			return fmt.Errorf("node list contains duplicate: %s", duplicate)
		}

		if !slices.Contains(nodeTypes, KubernetesNodeTypeServer) {
			return fmt.Errorf("cluster of only agent nodes cannot be formed")
		}
	}

	err := validateManifestURLs(&definition.Kubernetes)
	if err != nil {
		return fmt.Errorf("validating manifest urls: %w", err)
	}

	return nil
}

func validateManifestURLs(kubernetes *Kubernetes) error {
	if len(kubernetes.Manifests.URLs) == 0 {
		return nil
	}
	seenManifests := make(map[string]bool)

	for _, manifest := range kubernetes.Manifests.URLs {
		if !strings.HasPrefix(manifest, "http") {
			return fmt.Errorf("invalid manifest url, does not start with 'http://' or 'https://'")
		}

		if _, exists := seenManifests[manifest]; exists {
			return fmt.Errorf("duplicate manifest url found: '%s'", manifest)
		}

		seenManifests[manifest] = true
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

func validateUnattended(definition *Definition) error {
	if definition.Image.ImageType != TypeISO && definition.OperatingSystem.Unattended {
		return fmt.Errorf("unattended mode can only be used with image type '%s'", TypeISO)
	}

	if definition.Image.ImageType != TypeISO && definition.OperatingSystem.InstallDevice != "" {
		return fmt.Errorf("install device can only be selected with image type '%s'", TypeISO)
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

func validateEmbeddedArtifactRegistry(definition *Definition) error {
	if IsEmbeddedArtifactRegistryEmpty(definition.EmbeddedArtifactRegistry) {
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

func IsEmbeddedArtifactRegistryEmpty(registry EmbeddedArtifactRegistry) bool {
	return len(registry.HelmCharts) == 0 && len(registry.ContainerImages) == 0
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

func validatePackages(os *OperatingSystem) error {
	if duplicate := checkForDuplicates(os.Packages.PKGList); duplicate != "" {
		return fmt.Errorf("package list contains duplicate: %s", duplicate)
	}

	if duplicate := checkForDuplicates(os.Packages.AdditionalRepos); duplicate != "" {
		return fmt.Errorf("additional repository list contains duplicate: %s", duplicate)
	}

	if len(os.Packages.PKGList) > 0 && len(os.Packages.AdditionalRepos) == 0 && os.Packages.RegCode == "" {
		return fmt.Errorf("package list configured without providing additional repository or registration code")
	}

	return nil
}
