package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"go.uber.org/zap"
)

type Resolver interface {
	Resolve(string) (string, []string, error)
}

type resolver struct {
	// dir from where the resolver will work
	dir string
	// path to the image that the resolver will use as base
	imgPath string
	// type of the image that will be used as base (either ISO or RAW)
	imgType string
	// user provided packages for which dependency resolution will be done
	// podman client
	podman podman.Podman
}

// New creates a new Resolver instance that is based on the image context provided by the user
func New(ctx *image.Context) (Resolver, error) {
	resolverDir := filepath.Join(ctx.BuildDir, "resolver")
	if err := os.MkdirAll(resolverDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating %s dir: %w", resolverDir, err)
	}

	p, err := podman.New(resolverDir)
	if err != nil {
		return nil, fmt.Errorf("starting podman client: %w", err)
	}

	rpmPath := filepath.Join(ctx.ImageConfigDir, "rpms")
	if _, err := os.Stat(rpmPath); os.IsNotExist(err) {
		rpmPath = ""
	} else if err != nil {
		return nil, fmt.Errorf("validating rpm dir exists: %w", err)
	}

	return &resolver{
		dir:     resolverDir,
		imgPath: filepath.Join(ctx.ImageConfigDir, "images", ctx.ImageDefinition.Image.BaseImage),
		imgType: ctx.ImageDefinition.Image.ImageType,
		podman:  p,
	}, nil
}

// Resolve resolves all dependencies for the packages and third party rpms that have been configured by the user in the image context.
// It then outputs the set of resolved rpms to a directory from which an RPM repository can be created. Returns the full path to the created
// directory, the package/rpm names for which dependency resolution has been done, any errors that have occured.
//
// Parameters:
//   - out - location where the RPM directory will be created
func (r *resolver) Resolve(out string) (string, []string, error) {
	zap.L().Info("Resolving package dependencies...")

	if err := r.buildBase(); err != nil {
		return "", nil, fmt.Errorf("building base resolver image: %w", err)
	}

	return "", nil, nil
}
