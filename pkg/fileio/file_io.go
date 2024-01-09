package fileio

import (
	"errors"
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

	destFile, err := createFileWithPerms(dest, perms)
	if err != nil {
		return fmt.Errorf("creating file with permissions: %w", err)
	}

	defer func() {
		_ = destFile.Close()
	}()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}

func CopyFileN(src io.Reader, dest string, perms os.FileMode, n int64) error {
	destFile, err := createFileWithPerms(dest, perms)
	if err != nil {
		return fmt.Errorf("creating file with permissions: %w", err)
	}

	defer func() {
		_ = destFile.Close()
	}()

	for {
		// TODO: mitigate impact of a possible decompression
		// attack by doing a validation between copy chunks
		_, err := io.CopyN(destFile, src, n)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("copying file bytes: %w", err)
		}
	}
}

func createFileWithPerms(dest string, perms os.FileMode) (*os.File, error) {
	file, err := os.Create(dest)
	if err != nil {
		return nil, fmt.Errorf("creating file: %w", err)
	}

	if err = file.Chmod(perms); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("setting up permissions: %w", err)
	}

	return file, nil
}
