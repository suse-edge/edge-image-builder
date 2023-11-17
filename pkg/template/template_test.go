package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name           string
		templateName   string
		contents       string
		templateData   any
		expectedOutput string
		expectedErr    string
	}{
		{
			name:         "Template is successfully processed",
			templateName: "valid-template",
			contents:     "{{.Foo}} and {{.Bar}}",
			templateData: struct {
				Foo string
				Bar string
			}{
				Foo: "ooF",
				Bar: "raB",
			},
			expectedOutput: "ooF and raB",
		},
		{
			name:         "Templating fails due to missing data",
			templateName: "missing-data",
			contents:     "{{.Foo}} and {{.Bar}}",
			expectedErr:  "template data not provided",
		},
		{
			name:         "Templating fails due to invalid syntax",
			templateName: "invalid-syntax",
			contents:     "{{.Foo and ",
			templateData: struct{}{},
			expectedErr:  "parsing contents: template: invalid-syntax:1: unclosed action",
		},
		{
			name:         "Templating fails due to missing field",
			templateName: "invalid-data",
			contents:     "{{.Foo}} and {{.Bar}}",
			templateData: struct {
				Foo string
			}{
				Foo: "ooF",
			},
			expectedErr: "applying template: template: invalid-data:1:15: " +
				"executing \"invalid-data\" at <.Bar>: can't evaluate field Bar in type struct { Foo string }",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := Parse(test.templateName, test.contents, test.templateData)

			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
				assert.Equal(t, "", data)
			} else {
				require.Nil(t, err)
				assert.Equal(t, test.expectedOutput, data)
			}
		})
	}
}
