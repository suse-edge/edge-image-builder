package fileio

import (
	"fmt"
	"os"
	"text/template"
)

func WriteFile(filename string, contents string, templateData any) error {
	if templateData == nil {
		if err := os.WriteFile(filename, []byte(contents), os.ModePerm); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}

		return nil
	}

	tmpl, err := template.New(filename).Parse(contents)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if err = tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("applying template: %w", err)
	}

	return nil
}
