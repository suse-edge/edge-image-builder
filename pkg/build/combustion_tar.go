package build

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (g *Generator) generateTarball() error {
	if err := deleteFile(g.context.OutputPath()); err != nil {
		return fmt.Errorf("deleting existing combustion tarball: %w", err)
	}

	return g.createCompressedTarball()
}

func (g *Generator) createCompressedTarball() error {
	outputPath := g.context.OutputPath()

	directories := []string{
		g.context.CombustionDir,
		g.context.ArtefactsDir,
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("could not create combustion tarball: %w", err)
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, dir := range directories {
		err = addDirectoryToTar(tw, dir)
		if err != nil {
			return fmt.Errorf("could not add '%s' directory to tarball: %w", dir, err)
		}
	}
	return nil
}

func addDirectoryToTar(tw *tar.Writer, basePath string) error {
	baseDir := filepath.Dir(basePath)

	return filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walking '%s': %w", path, err)
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path for '%s': %w", path, err)
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("creating tar header for '%s': %w", path, err)
		}

		header.Name = relPath

		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return fmt.Errorf("reading symbolic link for '%s': %w", path, err)
			}
			header.Linkname = target
		}

		if err = tw.WriteHeader(header); err != nil {
			return fmt.Errorf("writing tar header for '%s': %w", path, err)
		}

		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("opening file '%s': %w", path, err)
			}
			defer file.Close()

			_, err = io.Copy(tw, file)
			if err != nil {
				return fmt.Errorf("copying file contents for '%s': %w", path, err)
			}
		}

		return nil
	})
}
