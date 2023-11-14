package fileio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileWithTemplate(t *testing.T) {
	// Setup
	tmpDir, err := os.MkdirTemp("", "eib-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	testData := "{{.Foo}} and {{.Bar}}"
	values := struct {
		Foo string
		Bar string
	}{
		Foo: "ooF",
		Bar: "raB",
	}
	testFilename := filepath.Join(tmpDir, "write-file-with-template.sh")

	// Test
	err = WriteFile(testFilename, testData, &values)

	// Verify
	require.NoError(t, err)

	foundData, err := os.ReadFile(testFilename)
	require.NoError(t, err)
	assert.Equal(t, "ooF and raB", string(foundData))
}
