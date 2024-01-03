package image

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestValidate(t *testing.T) {
	// Setup
	filename := "./testdata/full-valid-example.yaml"
	configData, err := os.ReadFile(filename)
	require.NoError(t, err)

	// Test
	definition, err := ParseDefinition(configData)
	_ = definition
	if err != nil {
		fmt.Println(err)
	}
}
