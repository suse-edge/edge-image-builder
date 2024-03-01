package fileio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.uber.org/zap"
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

// CopyFiles copies files from src to dest.
//
// If 'ext' is non-empty, copies only files with the specified extension, otherwise copies all files.
//
// If `copySubDir` is set to false, copies files only from 'src' directory
// and does not iterate over sub-directories.
//
// If `copySubDir` is set to true, iterates through all sub-directories
// and copies the directory tree along with all the files.
//
// If `copySubDir` is used with 'ext', iterates through all sub-directories
// and only copies files with the specified extension.
func CopyFiles(src, dest, ext string, copySubDir bool) error {
	files, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading source dir: %w", err)
	}

	if err = os.MkdirAll(dest, os.ModePerm); err != nil {
		return fmt.Errorf("creating directory '%s': %w", dest, err)
	}

	for _, file := range files {
		sourcePath := filepath.Join(src, file.Name())
		destPath := filepath.Join(dest, file.Name())

		if file.IsDir() {
			if !copySubDir {
				zap.S().Warnf("Skipping copy, '%s' is a directory", file.Name())
				continue
			}

			err = CopyFiles(sourcePath, destPath, ext, true)
			if err != nil {
				return fmt.Errorf("copying files from sub-directory '%s': %w", destPath, err)
			}
		} else {
			if ext != "" && filepath.Ext(file.Name()) != ext {
				zap.S().Debugf("Skipping %s as it is not a '%s' file", file.Name(), ext)
				continue
			}

			err := CopyFile(sourcePath, destPath, NonExecutablePerms)
			if err != nil {
				return fmt.Errorf("copying file %s: %w", sourcePath, err)
			}
		}
	}

	return nil
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
