package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestValidateElementalNoDir(t *testing.T) {
	ctx := image.Context{}

	failures := validateElemental(&ctx)
	assert.Len(t, failures, 0)
}

func TestValidateElemental(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))

	validElementalConfig := filepath.Join(elementalDir, "elemental_config.yaml")
	require.NoError(t, os.WriteFile(validElementalConfig, []byte(""), 0o600))

	tests := map[string]struct {
		ImageDefinition        *image.Definition
		ExpectedFailedMessages []string
	}{
		`valid, registration code no side-loaded rpms`: {
			ImageDefinition: &image.Definition{
				OperatingSystem: image.OperatingSystem{
					Packages: image.Packages{
						RegCode: "registration-code",
					},
				},
			},
		},
		`invalid, no registration code no side-loaded rpms`: {
			ImageDefinition: &image.Definition{},
			ExpectedFailedMessages: []string{
				"Operating system package registration code field must be defined when using Elemental or the " +
					"[elemental-register elemental-system-agent] RPMs must be manually side-loaded",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := image.Context{
				ImageConfigDir:  configDir,
				ImageDefinition: test.ImageDefinition,
			}
			failures := validateElemental(&ctx)
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

func TestValidateElementalConfigDirValid(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))

	elementalConfig := filepath.Join(elementalDir, "elemental_config.yaml")
	require.NoError(t, os.WriteFile(elementalConfig, []byte(""), 0o600))

	failures := validateElementalDir(elementalDir)
	assert.Len(t, failures, 0)
}

func TestValidateElementalConfigDirEmptyDir(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))

	failures := validateElementalDir(elementalDir)
	assert.Len(t, failures, 1)

	assert.Contains(t, failures[0].UserMessage, "Elemental config directory should not be present if it is empty")
}

func TestValidateElementalConfigDirMultipleFiles(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))

	firstElementalConfig := filepath.Join(elementalDir, "elemental_config1.yaml")
	require.NoError(t, os.WriteFile(firstElementalConfig, []byte(""), 0o600))
	secondElementalConfig := filepath.Join(elementalDir, "elemental_config2.yaml")
	require.NoError(t, os.WriteFile(secondElementalConfig, []byte(""), 0o600))

	failures := validateElementalDir(elementalDir)
	assert.Len(t, failures, 1)

	assert.Contains(t, failures[0].UserMessage, "Elemental config directory should only contain a singular 'elemental_config.yaml' file")
}

func TestValidateElementalConfigDirUnreadable(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))
	require.NoError(t, os.Chmod(elementalDir, 0o333))

	failures := validateElementalDir(elementalDir)
	assert.Len(t, failures, 1)

	assert.Contains(t, failures[0].UserMessage, "Elemental config directory could not be read")
}

func TestValidateElementalConfigurationManualRPMsNoRegistrationCode(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	ctx := &image.Context{
		ImageConfigDir:  configDir,
		ImageDefinition: &image.Definition{},
	}

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))

	elementalConfig := filepath.Join(elementalDir, "elemental_config.yaml")
	require.NoError(t, os.WriteFile(elementalConfig, []byte(""), 0o600))

	rpmDir := filepath.Join(configDir, "rpms")
	require.NoError(t, os.MkdirAll(rpmDir, os.ModePerm))

	elementalAgent := filepath.Join(rpmDir, "elemental-system-agent.rpm")
	require.NoError(t, os.WriteFile(elementalAgent, []byte(""), 0o600))

	elementalRegister := filepath.Join(rpmDir, "elemental-register.rpm")
	require.NoError(t, os.WriteFile(elementalRegister, []byte(""), 0o600))

	failures := validateElementalConfiguration(ctx)
	assert.Len(t, failures, 0)
}

func TestValidateElementalConfigurationManualRPMsWithRegistrationCode(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	ctx := &image.Context{
		ImageConfigDir: configDir,
		ImageDefinition: &image.Definition{
			OperatingSystem: image.OperatingSystem{
				Packages: image.Packages{
					RegCode: "registration-code",
				},
			},
		},
	}

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))

	elementalConfig := filepath.Join(elementalDir, "elemental_config.yaml")
	require.NoError(t, os.WriteFile(elementalConfig, []byte(""), 0o600))

	rpmDir := filepath.Join(configDir, "rpms")
	require.NoError(t, os.MkdirAll(rpmDir, os.ModePerm))

	elementalAgent := filepath.Join(rpmDir, "elemental-system-agent.rpm")
	require.NoError(t, os.WriteFile(elementalAgent, []byte(""), 0o600))

	elementalRegister := filepath.Join(rpmDir, "elemental-register.rpm")
	require.NoError(t, os.WriteFile(elementalRegister, []byte(""), 0o600))

	failures := validateElementalConfiguration(ctx)
	assert.Len(t, failures, 0)
}

func TestValidateElementalConfigurationManualRPMsMissingAgent(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	ctx := &image.Context{
		ImageConfigDir:  configDir,
		ImageDefinition: &image.Definition{},
	}

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))

	elementalConfig := filepath.Join(elementalDir, "elemental_config.yaml")
	require.NoError(t, os.WriteFile(elementalConfig, []byte(""), 0o600))

	rpmDir := filepath.Join(configDir, "rpms")
	require.NoError(t, os.MkdirAll(rpmDir, os.ModePerm))

	elementalRegister := filepath.Join(rpmDir, "elemental-register.rpm")
	require.NoError(t, os.WriteFile(elementalRegister, []byte(""), 0o600))

	failures := validateElementalConfiguration(ctx)
	assert.Len(t, failures, 1)

	assert.Contains(t, failures[0].UserMessage, "Not all of the necessary Elemental packages are provided, packages found: [elemental-register], packages missing: [elemental-system-agent]")
}

func TestValidateElementalConfigurationManualRPMsMissingRegister(t *testing.T) {
	configDir, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDir))
	}()

	ctx := &image.Context{
		ImageConfigDir:  configDir,
		ImageDefinition: &image.Definition{},
	}

	elementalDir := filepath.Join(configDir, "elemental")
	require.NoError(t, os.MkdirAll(elementalDir, os.ModePerm))

	elementalConfig := filepath.Join(elementalDir, "elemental_config.yaml")
	require.NoError(t, os.WriteFile(elementalConfig, []byte(""), 0o600))

	rpmDir := filepath.Join(configDir, "rpms")
	require.NoError(t, os.MkdirAll(rpmDir, os.ModePerm))

	elementalAgent := filepath.Join(rpmDir, "elemental-system-agent.rpm")
	require.NoError(t, os.WriteFile(elementalAgent, []byte(""), 0o600))

	failures := validateElementalConfiguration(ctx)
	assert.Len(t, failures, 1)

	assert.Contains(t, failures[0].UserMessage, "Not all of the necessary Elemental packages are provided, packages found: [elemental-system-agent], packages missing: [elemental-register]")
}
