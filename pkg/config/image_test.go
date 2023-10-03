package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	// Setup
	filename := "./testdata/valid_example.yaml"
	configData, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("Cannot read example YAML file: %v", filename)
	}

	// Test
	imageConfig, err := Parse(configData)
	if err != nil {
		t.Error("Parsing error: ", err)
	}

	// Verify
	assert.Equal(t, "1.0", imageConfig.ApiVersion)
	assert.Equal(t, "iso", imageConfig.ImageType)
}

func TestParseBadConfig(t *testing.T) {
	// Setup
	badData := []byte("Not actually YAML")

	// Test
	_, err := Parse(badData)

	// Verify
	assert.Error(t, err)
}
