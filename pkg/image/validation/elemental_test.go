package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

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
		`valid 1.1`: {
			ImageDefinition: &image.Definition{
				APIVersion: "1.1",
				OperatingSystem: image.OperatingSystem{
					Packages: image.Packages{
						RegCode: "registration-code",
					},
				},
			},
		},
		`1.1 no registration code`: {
			ImageDefinition: &image.Definition{
				APIVersion: "1.1",
			},
			ExpectedFailedMessages: []string{
				"Operating system package registration code field must be defined when using Elemental with SL Micro 6.0",
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

func TestValidateElementalConfigDir(t *testing.T) {
	configDirValid, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	configDirEmpty, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	configDirMultipleFiles, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	configDirInvalidName, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	configDirUnreadable, err := os.MkdirTemp("", "eib-config-")
	require.NoError(t, err)

	defer func() {
		assert.NoError(t, os.RemoveAll(configDirValid))
		assert.NoError(t, os.RemoveAll(configDirEmpty))
		assert.NoError(t, os.RemoveAll(configDirMultipleFiles))
		assert.NoError(t, os.RemoveAll(configDirInvalidName))
		assert.NoError(t, os.RemoveAll(configDirUnreadable))
	}()

	elementalDirValid := filepath.Join(configDirValid, "elemental")
	require.NoError(t, os.MkdirAll(elementalDirValid, os.ModePerm))

	validElementalConfig := filepath.Join(elementalDirValid, "elemental_config.yaml")
	require.NoError(t, os.WriteFile(validElementalConfig, []byte(""), 0o600))

	elementalDirEmpty := filepath.Join(configDirEmpty, "elemental")
	require.NoError(t, os.MkdirAll(elementalDirEmpty, os.ModePerm))

	elementalDirMultipleFiles := filepath.Join(configDirMultipleFiles, "elemental")
	require.NoError(t, os.MkdirAll(elementalDirMultipleFiles, os.ModePerm))

	firstElementalConfig := filepath.Join(elementalDirMultipleFiles, "elemental_config1.yaml")
	require.NoError(t, os.WriteFile(firstElementalConfig, []byte(""), 0o600))
	secondElementalConfig := filepath.Join(elementalDirMultipleFiles, "elemental_config2.yaml")
	require.NoError(t, os.WriteFile(secondElementalConfig, []byte(""), 0o600))

	elementalDirInvalidName := filepath.Join(configDirInvalidName, "elemental")
	require.NoError(t, os.MkdirAll(elementalDirInvalidName, os.ModePerm))

	invalidElementalConfig := filepath.Join(elementalDirInvalidName, "elemental.yaml")
	require.NoError(t, os.WriteFile(invalidElementalConfig, []byte(""), 0o600))

	elementalDirUnreadable := filepath.Join(configDirUnreadable, "elemental")
	require.NoError(t, os.MkdirAll(elementalDirUnreadable, os.ModePerm))
	require.NoError(t, os.Chmod(elementalDirUnreadable, 0o333))

	tests := map[string]struct {
		ExpectedFailedMessages []string
		ElementalDir           string
	}{
		`valid elemental dir`: {
			ElementalDir: elementalDirValid,
		},
		`empty elemental dir`: {
			ElementalDir: elementalDirEmpty,
			ExpectedFailedMessages: []string{
				"Elemental config directory should not be present if it is empty",
			},
		},
		`multiple files in elemental dir`: {
			ElementalDir: elementalDirMultipleFiles,
			ExpectedFailedMessages: []string{
				"Elemental config directory should only contain a singular 'elemental_config.yaml' file",
			},
		},
		`invalid name in elemental dir`: {
			ElementalDir: elementalDirInvalidName,
			ExpectedFailedMessages: []string{
				"Elemental config file should only be named `elemental_config.yaml`",
			},
		},
		`unreadable elemental dir`: {
			ElementalDir: elementalDirUnreadable,
			ExpectedFailedMessages: []string{
				fmt.Sprintf("Elemental config directory could not be read: open %s: permission denied", elementalDirUnreadable),
				"Elemental config directory should not be present if it is empty",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			failures := validateElementalDir(test.ElementalDir)
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
