package combustion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"gopkg.in/yaml.v3"
)

func TestWriteElementalScriptFile(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		Elemental: image.Elemental{
			Registration: image.ElementalRegistration{
				RegistrationURL: "https://example.com/registration",
				CACert:          "ca-cert.pem",
				EmulateTPM:      true,
				EmulateTPMSeed:  1,
				AuthType:        "tpm",
			},
		},
	}

	// Test
	scripts, err := configureElemental(ctx)

	// Verify
	require.NoError(t, err)

	require.Len(t, scripts, 1)

	configFilename := filepath.Join(ctx.CombustionDir, elementalConfigName)
	_, err = os.Stat(configFilename)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(configFilename)
	require.NoError(t, err)

	var foundDefinition image.Definition
	err = yaml.Unmarshal(foundBytes, &foundDefinition)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/registration", foundDefinition.Elemental.Registration.RegistrationURL)
	assert.Equal(t, "ca-cert.pem", foundDefinition.Elemental.Registration.CACert)
	assert.Equal(t, true, foundDefinition.Elemental.Registration.EmulateTPM)
	assert.Equal(t, 1, foundDefinition.Elemental.Registration.EmulateTPMSeed)
	assert.Equal(t, "tpm", foundDefinition.Elemental.Registration.AuthType)
}

func TestWriteElementalCombustionScript(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	// Test
	err := writeElementalCombustionScript(ctx)

	// Verify
	require.NoError(t, err)

	scriptFilename := filepath.Join(ctx.CombustionDir, elementalScriptName)
	_, err = os.Stat(scriptFilename)
	require.NoError(t, err)

	foundBytes, err := os.ReadFile(scriptFilename)
	require.NoError(t, err)
	found := string(foundBytes)
	assert.Contains(t, found, "elemental-register --config-path /etc/elemental/config.yaml")
}
