package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/podman"
	"github.com/suse-edge/edge-image-builder/pkg/rpm"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	resolverImageRef = "pkg-resolver"
	dockerfileName   = "Dockerfile"
	rpmRepoName      = "rpm-repo"
)

//go:embed scripts/Dockerfile.tpl
var dockerfileTemplate string

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
	packages *image.Packages
	// user provided directory containing additional rpms for which dependency resolution will be done
	customRPMDir string
	// podman client
	podman podman.Podman
	// hepler property, contains the names of the rpms that have been taken from the customRPMDir
	rpms []string
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
		dir:          resolverDir,
		imgPath:      filepath.Join(ctx.ImageConfigDir, "images", ctx.ImageDefinition.Image.BaseImage),
		imgType:      ctx.ImageDefinition.Image.ImageType,
		packages:     &ctx.ImageDefinition.OperatingSystem.Packages,
		podman:       p,
		customRPMDir: rpmPath,
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

	if err := r.prepare(); err != nil {
		return "", nil, fmt.Errorf("generating context for the resolver image: %w", err)
	}

	if err := r.podman.Build(r.generateBuildContextPath(), resolverImageRef); err != nil {
		return "", nil, fmt.Errorf("building resolver image: %w", err)
	}

	id, err := r.podman.Run(resolverImageRef)
	if err != nil {
		return "", nil, fmt.Errorf("run container from resolver image %s: %w", resolverImageRef, err)
	}

	err = r.podman.Copy(id, r.generateRPMRepoPath(), out)
	if err != nil {
		return "", nil, fmt.Errorf("copying resolved pkg cache to %s: %w", out, err)
	}

	return filepath.Join(out, rpmRepoName), r.getPKGInstallList(), nil
}

func (r *resolver) prepare() error {
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

func (r *resolver) writeDockerfile() error {
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
		AddRepo:   strings.Join(r.packages.AddRepos, " "),
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

func (r *resolver) getPKGForResolve() []string {
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

func (r *resolver) getPKGInstallList() []string {
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
func (r *resolver) generateBuildContextPath() string {
	return filepath.Join(r.dir, "build")
}

// path to the rpms directory, as seen in the EIB image
func (r *resolver) generateRPMPathInBuildContext() string {
	return filepath.Join(r.generateBuildContextPath(), "rpms")
}

// path to rpm cache dir, as seen in the resolver image
func (r *resolver) generateRPMRepoPath() string {
	return filepath.Join(os.TempDir(), rpmRepoName)
}

// path to the dir containing local rpms, as seen in the resolver image
func (r *resolver) generateLocalRPMDirPath() string {
	return filepath.Join(r.generateRPMRepoPath(), "local")
}
