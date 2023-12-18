package podman

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

func generatePodmanLogFile(fileName, out string) (*os.File, error) {
	timestamp := time.Now().Format("Jan02_15-04-05")
	filename := fmt.Sprintf(fileName, timestamp)
	logFilename := filepath.Join(out, filename)

	logFile, err := os.Create(logFilename)
	if err != nil {
		return nil, fmt.Errorf("creating podman log file: %w", err)
	}
	zap.L().Sugar().Debugf("podman log file created: %s", logFilename)

	return logFile, err
}

func untar(arch, dest string) error {
	reader, err := os.Open(arch)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer reader.Close()

	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return fmt.Errorf("reading archive: %w", err)
		}

		path, err := sanitizedPath(dest, header.Name)
		if err != nil {
			return fmt.Errorf("illegal file path: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err = os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					if err = os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
						return fmt.Errorf("creating directory %s: %w", path, err)
					}
				} else {
					return fmt.Errorf("checking directory %s: %w", path, err)
				}
			}
		case tar.TypeReg:
			if err = copyFile(path, os.FileMode(header.Mode), tarReader); err != nil {
				return fmt.Errorf("copying file: %w", err)
			}
		default:
			return fmt.Errorf("unexpected header type %b: %w", header.Typeflag, err)
		}
	}
}

func copyFile(path string, mode os.FileMode, reader io.Reader) error {
	var file *os.File
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", path, err)
	}
	defer file.Close()

	if err = fileCopyN(file, reader, 4096); err != nil {
		return fmt.Errorf("copying files to %s: %w", path, err)
	}

	return nil
}

// copy file at N byte sections (gosec G110)
func fileCopyN(w io.Writer, r io.Reader, n int64) error {
	for {
		_, err := io.CopyN(w, r, n)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("copying file bytes: %w", err)
		}
	}
	return nil
}

// make sure that path is legal and not tainted (gosec G305)
func sanitizedPath(dest, fileName string) (string, error) {
	path := filepath.Join(dest, fileName)
	if strings.HasPrefix(path, filepath.Clean(dest)) {
		return path, nil
	}

	return "", fmt.Errorf("content filepath is tainted: %s", path)
}
