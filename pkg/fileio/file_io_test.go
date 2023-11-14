package fileio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile(t *testing.T) {
	const tmpDir = "write-test"

	require.NoError(t, os.Mkdir(tmpDir, os.ModePerm))
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name             string
		filename         string
		contents         string
		templateData     any
		expectedContents string
		expectedErr      string
	}{
		{
			name:             "Standard file is successfully written",
			filename:         "standard",
			contents:         "this is a non-templated file",
			expectedContents: "this is a non-templated file",
		},
		{
			name:     "Templated file is successfully written",
			filename: "template",
			contents: "{{.Foo}} and {{.Bar}}",
			templateData: struct {
				Foo string
				Bar string
			}{
				Foo: "ooF",
				Bar: "raB",
			},
			expectedContents: "ooF and raB",
		},
		{
			name:         "Templated file is not written due to invalid syntax",
			filename:     "invalid-syntax",
			contents:     "{{.Foo and ",
			templateData: struct{}{},
			expectedErr:  "parsing template: template: write-test/invalid-syntax:1: unclosed action",
		},
		{
			name:     "Templated file is not written due to missing field",
			filename: "invalid-data",
			contents: "{{.Foo}} and {{.Bar}}",
			templateData: struct {
				Foo string
			}{
				Foo: "ooF",
			},
			expectedErr: "applying template: template: write-test/invalid-data:1:15: " +
				"executing \"write-test/invalid-data\" at <.Bar>: can't evaluate field Bar in type struct { Foo string }",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filename := filepath.Join(tmpDir, test.filename)

			err := WriteFile(filename, test.contents, test.templateData)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)

				contents, err := os.ReadFile(filename)
				require.NoError(t, err)

				assert.Equal(t, test.expectedContents, string(contents))
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	const (
		source  = "file_io.go" // use the source code file as a valid input
		destDir = "copy-test"
	)

	require.NoError(t, os.Mkdir(destDir, os.ModePerm))
	defer os.RemoveAll(destDir)

	tests := []struct {
		name        string
		source      string
		destination string
		expectedErr string
	}{
		{
			name:        "Source file does not exist",
			source:      "<missing>",
			expectedErr: "opening source file: open <missing>: no such file or directory",
		},
		{
			name:        "Destination is an empty file",
			source:      source,
			destination: "",
			expectedErr: "creating destination file: open : no such file or directory",
		},
		{
			name:        "Destination is a directory",
			source:      source,
			destination: "copy-test/",
			expectedErr: "creating destination file: open copy-test/: is a directory",
		},
		{
			name:        "File is successfully copied",
			source:      source,
			destination: "copy-test/copy.go",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CopyFile(test.source, test.destination)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)

				src, err := os.ReadFile(test.source)
				require.NoError(t, err)

				dest, err := os.ReadFile(test.destination)
				require.NoError(t, err)

				assert.Equal(t, src, dest)
			}
		})
	}
}
