package template

import (
	"bytes"
	"fmt"
	"text/template"
)

func Parse(name string, contents string, templateData any) (string, error) {
	if templateData == nil {
		return "", fmt.Errorf("template data not provided")
	}

	tmpl, err := template.New(name).Parse(contents)
	if err != nil {
		return "", fmt.Errorf("parsing contents: %w", err)
	}

	var buff bytes.Buffer
	if err = tmpl.Execute(&buff, templateData); err != nil {
		return "", fmt.Errorf("applying template: %w", err)
	}

	return buff.String(), nil
}
