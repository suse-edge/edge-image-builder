package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	resolverImageRef = "pkg-resolver"
	dockerfileName   = "Dockerfile"
	rpmRepoName      = "rpm-repo"
)

//go:embed templates/Dockerfile.tpl
var dockerfileTemplate string

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
	// helper property, contains the names of the rpms that have been taken from the customRPMDir
	rpms []string
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

// Resolve resolves all dependencies for the packages and third party rpms that have been configured by the user in the image context.
// It then outputs the set of resolved rpms to a directory from which an RPM repository can be created. Returns the full path to the created
// directory, the package/rpm names for which dependency resolution has been done, or an error if one has occurred.
//
// Parameters:
//   - out - location where the RPM directory will be created
func (r *Resolver) Resolve(out string, podman image.Podman) (rpmDir string, pkgList []string, err error) {
	zap.L().Info("Resolving package dependencies...")

	if err = r.buildBase(podman); err != nil {
		return "", nil, fmt.Errorf("building base resolver image: %w", err)
	}

	if err = r.prepare(); err != nil {
		return "", nil, fmt.Errorf("generating context for the resolver image: %w", err)
	}

	if err = podman.Build(r.generateBuildContextPath(), resolverImageRef); err != nil {
		return "", nil, fmt.Errorf("building resolver image: %w", err)
	}

	id, err := podman.Create(resolverImageRef)
	if err != nil {
		return "", nil, fmt.Errorf("run container from resolver image %s: %w", resolverImageRef, err)
	}

	err = podman.Copy(id, r.generateRPMRepoPath(), out)
	if err != nil {
		return "", nil, fmt.Errorf("copying resolved package cache to %s: %w", out, err)
	}

	return filepath.Join(out, rpmRepoName), r.getPKGInstallList(), nil
}

func (r *Resolver) prepare() error {
	zap.L().Info("Preparing resolver image context...")

	buildContext := r.generateBuildContextPath()
	if err := os.MkdirAll(buildContext, os.ModePerm); err != nil {
		return fmt.Errorf("creating build context dir %s: %w", buildContext, err)
	}

	if r.customRPMDir != "" {
		dest := r.generateRPMPathInBuildContext()
		if err := os.MkdirAll(dest, os.ModePerm); err != nil {
			return fmt.Errorf("creating rpm directory in resolver dir: %w", err)
		}

		rpmNames, err := rpm.CopyRPMs(r.customRPMDir, dest)
		if err != nil {
			return fmt.Errorf("copying local rpms to %s: %w", dest, err)
		}
		r.rpms = rpmNames
	}

	if err := r.writeDockerfile(); err != nil {
		return fmt.Errorf("writing dockerfile: %w", err)
	}

	zap.L().Info("Resolver image context setup successful")
	return nil
}

func (r *Resolver) writeDockerfile() error {
	values := struct {
		BaseImage   string
		RegCode     string
		AddRepo     string
		CacheDir    string
		PkgList     string
		FromRPMPath string
		ToRPMPath   string
	}{
		BaseImage: baseImageRef,
		RegCode:   r.packages.RegCode,
		AddRepo:   strings.Join(r.packages.AdditionalRepos, " "),
		CacheDir:  r.generateRPMRepoPath(),
		PkgList:   strings.Join(r.getPKGForResolve(), " "),
	}

	if r.customRPMDir != "" {
		values.FromRPMPath = filepath.Base(r.generateRPMPathInBuildContext())
		values.ToRPMPath = r.generateLocalRPMDirPath()
	}

	data, err := template.Parse(dockerfileName, dockerfileTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", dockerfileName, err)
	}

	filename := filepath.Join(r.generateBuildContextPath(), dockerfileName)
	if err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

func (r *Resolver) getPKGForResolve() []string {
	list := []string{}

	if len(r.packages.PKGList) > 0 {
		list = append(list, r.packages.PKGList...)
	}

	if len(r.rpms) > 0 {
		// generate RPM paths as seen in the resolver image,
		// needed so that 'zypper install' can locate the rpms
		for _, name := range r.rpms {
			list = append(list, filepath.Join(r.generateLocalRPMDirPath(), name))
		}
	}
	return list
}

func (r *Resolver) getPKGInstallList() []string {
	list := []string{}

	if len(r.packages.PKGList) > 0 {
		list = append(list, r.packages.PKGList...)
	}

	if len(r.rpms) > 0 {
		// generate the RPMs as package names,
		// so that zypper can locate them in the RPM repository
		for _, name := range r.rpms {
			list = append(list, strings.TrimSuffix(name, filepath.Ext(name)))
		}
	}
	return list
}

// path to the build dir, as seen in the EIB image
func (r *Resolver) generateBuildContextPath() string {
	return filepath.Join(r.dir, "build")
}

// path to the rpms directory in the resolver build context, as seen in the EIB image
func (r *Resolver) generateRPMPathInBuildContext() string {
	return filepath.Join(r.generateBuildContextPath(), "rpms")
}

// path to rpm cache dir, as seen in the resolver image
func (r *Resolver) generateRPMRepoPath() string {
	return filepath.Join(os.TempDir(), rpmRepoName)
}

// path to the dir containing local rpms, as seen in the resolver image
func (r *Resolver) generateLocalRPMDirPath() string {
	return filepath.Join(r.generateRPMRepoPath(), "local")
}
