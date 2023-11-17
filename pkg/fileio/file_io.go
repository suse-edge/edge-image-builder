package fileio

import (
	"fmt"
	"io"
	"os"
)

const (
	// ExecutablePerms are Linux permissions (rwxr--r--) for executable files (scripts, binaries, etc.)
	ExecutablePerms os.FileMode = 0o744
	// NonExecutablePerms are Linux permissions (rw-r--r--) for non-executable files (configs, RPMs, etc.)
	NonExecutablePerms os.FileMode = 0o644
)

func CopyFile(src string, dest string, perms os.FileMode) error {
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

	if err = destFile.Chmod(perms); err != nil {
		return fmt.Errorf("adjusting permissions: %w", err)
	}

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}
