package podman

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/buildah/define"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/containers/podman/v4/pkg/bindings/containers"
	"github.com/containers/podman/v4/pkg/bindings/images"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/containers/podman/v4/pkg/specgen"
	"go.uber.org/zap"
)

const (
	podmanSock         = "unix:///run/podman/podman.sock"
	dockerfile         = "Dockerfile"
	podmanDirName      = "podman"
	podmanBuildLogFile = "podman-image-build-%s.log"
)

type Podman interface {
	Import(tarball, ref string) error
	Build(context, name string) error
	Run(img string) (string, error)
	Copy(id, src, dest string) error
}

type podman struct {
	context context.Context
	socket  string
	out     string
}

// New setups a podman listening service and returns a connected podman client.
//
// Parameters:
//   - out - location for podman to output any logs created as a result of podman commands
func New(out string) (Podman, error) {
	podmanDirPath := filepath.Join(out, podmanDirName)
	if err := os.MkdirAll(podmanDirPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating %s dir: %w", podmanDirPath, err)
	}

	if err := setupAPIListener(podmanDirPath); err != nil {
		return nil, fmt.Errorf("creating new podman instance: %w", err)
	}

	conn, err := bindings.NewConnection(context.Background(), podmanSock)
	if err != nil {
		return nil, fmt.Errorf("creating new podman connection: %w", err)
	}

	return &podman{
		context: conn,
		socket:  podmanSock,
		out:     podmanDirPath,
	}, nil
}

// Import imports a tarball and saves it as a filesystem image
//
// Parametes:
//   - tarball - path to the tarball to be imported
//   - ref 	  - name for the image that will be created from the tarball
func (p *podman) Import(tarball, ref string) error {
	zap.L().Sugar().Infof("Importing image '%s' from tarball...", ref)
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
//
// Parameters:
//   - context - context from where the image will be built
//   - name    - name for the image
func (p *podman) Build(context, name string) error {
	zap.L().Sugar().Infof("Building image %s...", name)
	logFile, err := generatePodmanLogFile(podmanBuildLogFile, p.out)
	if err != nil {
		return fmt.Errorf("generating podman build log file: %w", err)
	}

	eOpts := entities.BuildOptions{
		BuildOptions: define.BuildOptions{
			ContextDirectory: context,
			Output:           name,
			Out:              logFile,
			Err:              logFile,
		},
	}

	_, err = images.Build(p.context, []string{dockerfile}, eOpts)
	if err != nil {
		return fmt.Errorf("building image from context %s: %w", context, err)
	}

	return nil
}

// Run runs a container from the given image. Returns the id of the container.
func (p *podman) Run(img string) (string, error) {
	zap.L().Sugar().Infof("Running container from %s image...", img)

	s := specgen.NewSpecGenerator(img, false)
	createResponse, err := containers.CreateWithSpec(p.context, s, nil)
	if err != nil {
		return "", fmt.Errorf("creating container with sepc %v: %w", s, err)
	}

	if err := containers.Start(p.context, createResponse.ID, nil); err != nil {
		return "", fmt.Errorf("starting container with sepc %v: %w", s, err)
	}

	return createResponse.ID, nil
}

// Copy copies a file or directory from a source located in the container
// to a destination located outside of the container.
//
// Note: No need to create a placeholder file/directory in dest. The file/directory will be created
// automatically upon copying from source.
func (p *podman) Copy(id, src, dest string) error {
	zap.L().Sugar().Infof("Copying %s from container %s to %s", src, id, dest)

	tmpArch, err := os.Create(filepath.Join(os.TempDir(), "tmp.tar"))
	if err != nil {
		return fmt.Errorf("creating podman log file: %w", err)
	}

	defer os.RemoveAll(tmpArch.Name())

	copyFunc, err := containers.CopyToArchive(p.context, id, src, tmpArch)
	if err != nil {
		return fmt.Errorf("creating copy function for archive from %s: %w", src, err)
	}

	if err := copyFunc(); err != nil {
		return fmt.Errorf("copying archive from %s to %s: %w", src, dest, err)
	}

	if err := untar(tmpArch.Name(), dest); err != nil {
		return fmt.Errorf("extracting archive to %s: %w", dest, err)
	}

	return nil
}
