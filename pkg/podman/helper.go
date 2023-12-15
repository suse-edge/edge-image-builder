package podman

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("reading archive: %w", err)
		}

		path := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
						return fmt.Errorf("creating directory %s: %w", path, err)
					}
				} else {
					return fmt.Errorf("checking directory %s: %w", path, err)
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("opening file %s: %w", path, err)
			}
			defer f.Close()

			if _, err := io.Copy(f, tarReader); err != nil {
				return fmt.Errorf("copying files to %s: %w", path, err)
			}
		default:
			return fmt.Errorf("unexpected header type %b: %w", header.Typeflag, err)
		}
	}
}
