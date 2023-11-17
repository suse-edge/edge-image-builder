package fileio

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile(t *testing.T) {
	const tmpDirPrefix = "eib-write-file-test-"

	tmpDir, err := os.MkdirTemp("", tmpDirPrefix)
	require.NoError(t, err)
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filename := filepath.Join(tmpDir, test.filename)

			err := WriteFile(filename, test.contents)

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

func TestWriteTemplate(t *testing.T) {
	const tmpDirPrefix = "eib-write-template-test-"

	tmpDir, err := os.MkdirTemp("", tmpDirPrefix)
	require.NoError(t, err)
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
			name:        "Templated file is not written due to missing data",
			filename:    "missing-data",
			contents:    "{{.Foo}} and {{.Bar}}",
			expectedErr: "template data not provided",
		},
		{
			name:         "Templated file is not written due to invalid syntax",
			filename:     "invalid-syntax",
			contents:     "{{.Foo and ",
			templateData: struct{}{},
			expectedErr:  fmt.Sprintf("parsing template: template: %s/invalid-syntax:1: unclosed action", tmpDir),
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
			expectedErr: fmt.Sprintf("applying template: template: %[1]s/invalid-data:1:15: "+
				"executing \"%[1]s/invalid-data\" at <.Bar>: can't evaluate field Bar in type struct { Foo string }", tmpDir),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filename := filepath.Join(tmpDir, test.filename)

			err := WriteTemplate(filename, test.contents, test.templateData)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				require.Nil(t, err)

				contents, err := os.ReadFile(filename)
				require.NoError(t, err)
				assert.Equal(t, test.expectedContents, string(contents))

				info, err := os.Stat(filename)
				require.NoError(t, err)
				assert.Equal(t, fs.FileMode(0o744), info.Mode())
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	const (
		source        = "file_io.go" // use the source code file as a valid input
		destDirPrefix = "eib-copy-file-test-"
	)

	tmpDir, err := os.MkdirTemp("", destDirPrefix)
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

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
			destination: tmpDir,
			expectedErr: fmt.Sprintf("creating destination file: open %s: is a directory", tmpDir),
		},
		{
			name:        "File is successfully copied",
			source:      source,
			destination: fmt.Sprintf("%s/copy.go", tmpDir),
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
