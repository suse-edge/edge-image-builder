package fileio

import (
	"fmt"
	"os"
	"text/template"
)

func WriteFile(filename string, contents string, templateData any) error {
	if templateData != nil {
		tmpl, err := template.New(filename).Parse(contents)
		if err != nil {
			return fmt.Errorf("creating template for file %s: %w", filename, err)
		}

		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("creating file %s: %w", filename, err)
		}
		defer file.Close()

		err = tmpl.Execute(file, templateData)
		if err != nil {
			return fmt.Errorf("applying the template at %s: %w", filename, err)
		}
	} else {
		err := os.WriteFile(filename, []byte(contents), os.ModePerm)
		if err != nil {
			return fmt.Errorf("writing file %s: %w", filename, err)
		}
	}
	return nil
}
