package fileio

import (
	"fmt"
	"io"
	"os"
	"text/template"
)

const (
	// ExecutablePerms are Linux permissions (rwxr--r--) for executable files (scripts, binaries, etc.)
	ExecutablePerms os.FileMode = 0o744
	// NonExecutablePerms are Linux permissions (rw-r--r--) for non-executable files (configs, RPMs, etc.):
	NonExecutablePerms os.FileMode = 0o644
)

func WriteTemplate(filename string, contents string, templateData any) error {
	if templateData == nil {
		return fmt.Errorf("template data not provided")
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

	if err = file.Chmod(ExecutablePerms); err != nil {
		return fmt.Errorf("applying executable permissions: %w", err)
	}

	if err = tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("applying template: %w", err)
	}

	return nil
}

func CopyFile(src string, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer func() {
		_ = destFile.Close()
	}()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}
