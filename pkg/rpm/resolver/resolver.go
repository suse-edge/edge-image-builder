package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type Resolver struct {
	// dir from where the resolver will work
	dir string
	// path to the image that the resolver will use as base
	imgPath string
	// type of the image that will be used as base (either ISO or RAW)
	imgType string
	// user provided packages for which dependency resolution will be done
	packages *image.Packages
	// user provided directory containing additional rpms for which dependency resolution will be done
	customRPMDir string
}

// New creates a new Resolver instance that is based on the image context provided by the user
func New(buildDir, imgConfDir string, imageDef *image.Definition) (*Resolver, error) {
	rpmPath := filepath.Join(imgConfDir, "rpms")
	if _, err := os.Stat(rpmPath); os.IsNotExist(err) {
		rpmPath = ""
	} else if err != nil {
		return nil, fmt.Errorf("validating rpm dir exists: %w", err)
	}

	return &Resolver{
		dir:          buildDir,
		imgPath:      filepath.Join(imgConfDir, "images", imageDef.Image.BaseImage),
		imgType:      imageDef.Image.ImageType,
		packages:     &imageDef.OperatingSystem.Packages,
		customRPMDir: rpmPath,
	}, nil
}

func (r *Resolver) Resolve(out string, podman image.Podman) (rpmDir string, pkgList []string, err error) {
	return "", nil, nil
}
