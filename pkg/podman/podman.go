package podman

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/buildah/define"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/containers/podman/v4/pkg/specgen"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"go.uber.org/zap"
)

const (
	podmanSocketURI    = "unix://%s"
	dockerfile         = "Dockerfile"
	podmanDirName      = "podman"
	podmanBuildLogFile = "podman-image-build.log"
)

type Podman struct {
	context context.Context
	out     string
}

// New setups a podman listening service and returns a connected podman client.
//
// Parameters:
//   - out - location for podman to output any logs created as a result of podman commands
func New(out string) (*Podman, error) {
	if err := setupAPIListener(out); err != nil {
		return nil, fmt.Errorf("creating new podman instance: %w", err)
	}

	conn, err := bindings.NewConnection(context.Background(), fmt.Sprintf(podmanSocketURI, podmanSocketPath))
	if err != nil {
		return nil, fmt.Errorf("creating new podman connection: %w", err)
	}

	return &Podman{
		context: conn,
		out:     out,
	}, nil
}

// Import imports a tarball and saves it as a filesystem image
//
// Parameters:
//   - tarball - path to the tarball to be imported
//   - ref 	  - name for the image that will be created from the tarball
func (p *Podman) Import(tarball, ref string) error {
	zap.S().Infof("Importing image '%s' from tarball...", ref)
	f, err := os.Open(tarball)
	if err != nil {
		return fmt.Errorf("opening tarball %s: %w", tarball, err)
	}
	_, err = images.Import(p.context, f, &images.ImportOptions{Reference: &ref})
	if err != nil {
		return fmt.Errorf("importing tarball %s: %w", tarball, err)
	}

	return nil
}

// Build looks for a 'Dockerfile' in the given context and build a podman image
// from it.
func (p *Podman) Build(imageContext, imageName string) error {
	zap.S().Infof("Building image %s...", imageName)

	logFile, err := os.Create(filepath.Join(p.out, podmanBuildLogFile))
	if err != nil {
		return fmt.Errorf("generating podman build log file: %w", err)
	}
	defer logFile.Close()

	eOpts := entities.BuildOptions{
		BuildOptions: define.BuildOptions{
			ContextDirectory: imageContext,
			Output:           imageName,
			Out:              logFile,
			Err:              logFile,
		},
	}

	_, err = images.Build(p.context, []string{dockerfile}, eOpts)
	if err != nil {
		return fmt.Errorf("building image from context %s: %w", imageContext, err)
	}

	return nil
}

// Create creates a container from the given image. Returns the id of the container.
func (p *Podman) Create(img string) (string, error) {
	zap.S().Infof("Creating container from %s image...", img)

	s := specgen.NewSpecGenerator(img, false)
	createResponse, err := containers.CreateWithSpec(p.context, s, nil)
	if err != nil {
		return "", fmt.Errorf("creating container with spec %v: %w", s, err)
	}

	return createResponse.ID, nil
}

// Copy copies a file or directory from a source located in the container
// to a destination located outside of the container.
//
// Note: No need to create a placeholder file/directory in dest. The file/directory will be created
// automatically upon copying from source.
func (p *Podman) Copy(id, src, dest string) error {
	zap.S().Infof("Copying %s from container %s to %s", src, id, dest)

	reader, writer := io.Pipe()
	defer reader.Close()

	copyFunc, err := containers.CopyToArchive(p.context, id, src, writer)
	if err != nil {
		return fmt.Errorf("creating copy function for archive from %s: %w", src, err)
	}

	go func() {
		err := copyFunc()
		if err != nil {
			zap.S().Errorf("Copying %s to %s failed: %s", src, dest, err)
		}

		writer.Close()
	}()

	if err := untar(reader, dest); err != nil {
		return fmt.Errorf("extracting archive to %s: %w", dest, err)
	}

	return nil
}

func untar(arch io.Reader, dest string) error {
	const (
		chunkSize = 4096
	)

	tarReader := tar.NewReader(arch)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("reading archive: %w", err)
		}

		path, err := sanitizedPath(dest, header.Name)
		if err != nil {
			return fmt.Errorf("illegal file path: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("creating directory %s: %w", path, err)
			}
		case tar.TypeReg:
			if err = fileio.CopyFileN(tarReader, path, os.FileMode(header.Mode), chunkSize); err != nil {
				return fmt.Errorf("copying file: %w", err)
			}
		default:
			return fmt.Errorf("unexpected header type %b", header.Typeflag)
		}
	}
}

// make sure that path is legal and not tainted (gosec G305)
func sanitizedPath(dest, fileName string) (string, error) {
	path := filepath.Join(dest, fileName)
	if strings.HasPrefix(path, filepath.Clean(dest)) {
		return path, nil
	}

	return "", fmt.Errorf("content filepath is tainted: %s", path)
}
