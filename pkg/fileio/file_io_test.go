package fileio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile(t *testing.T) {
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
			tmpDir := "write-test"
			require.NoError(t, os.Mkdir(tmpDir, os.ModePerm))
			defer os.RemoveAll(tmpDir)

			filename := filepath.Join(tmpDir, test.filename)
			err := WriteFile(filename, test.contents, test.templateData)
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				return
			}

			require.Nil(t, err)

			contents, err := os.ReadFile(filename)
			require.NoError(t, err)

			assert.Equal(t, test.expectedContents, string(contents))
		})
	}
}
