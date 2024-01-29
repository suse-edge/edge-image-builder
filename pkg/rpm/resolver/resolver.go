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

type Podman interface {
	Import(tarball, ref string) error
	Build(context, name string) error
	Create(img string) (string, error)
	Copy(id, src, dest string) error
}

type Resolver struct {
	// dir from where the resolver will work
	dir string
	// path to the image that the resolver will use as base
	imgPath string
	// type of the image that will be used as base (either ISO or RAW)
	imgType string
	// podman client which to use for container management tasks
	podman Podman
	// helper property, contains the names of the rpms that have been taken from the customRPMDir
	rpms []string
}

func New(workDir, imgPath, imgType string, podman Podman) *Resolver {
	return &Resolver{
		dir:     workDir,
		imgPath: imgPath,
		imgType: imgType,
		podman:  podman,
	}
}

// Resolve resolves all dependencies for the provided pacakges and rpms. It then outputs the set of resolved rpms to a
// directory (located in the provdied 'outputDir') from which an RPM repository can be created.
//
// Returns the full path to the created directory, the package/rpm names for which dependency resolution has been done, or an error if one has occurred.
//
// Parameters:
// - packages - pacakge configuration
//
// - localPackagesPath - path to a directory containing local rpm packages. Will not be considered if left empty.
//
// - outputDir - directory in which the resolver will create a directory containing the resolved rpms.
func (r *Resolver) Resolve(packages *image.Packages, localPackagesPath, outputDir string) (rpmDirPath string, pkgList []string, err error) {
	zap.L().Info("Resolving package dependencies...")

	if err = r.buildBase(); err != nil {
		return "", nil, fmt.Errorf("building base resolver image: %w", err)
	}

	if err = r.prepare(localPackagesPath, packages); err != nil {
		return "", nil, fmt.Errorf("generating context for the resolver image: %w", err)
	}

	if err = r.podman.Build(r.generateBuildContextPath(), resolverImageRef); err != nil {
		return "", nil, fmt.Errorf("building resolver image: %w", err)
	}

	id, err := r.podman.Create(resolverImageRef)
	if err != nil {
		return "", nil, fmt.Errorf("run container from resolver image %s: %w", resolverImageRef, err)
	}

	err = r.podman.Copy(id, r.generateResolverImgRPMRepoPath(), outputDir)
	if err != nil {
		return "", nil, fmt.Errorf("copying resolved package cache to %s: %w", outputDir, err)
	}

	// rpmRepoName is the name of the directory to which all packages/rpms have been resovled to.
	// Since we are copying a directory inside of the 'outputDir', we concatenate the path in order
	// to return the correct path.
	return filepath.Join(outputDir, rpmRepoName), r.generatePKGInstallList(packages), nil
}

func (r *Resolver) prepare(localPackagesPath string, packages *image.Packages) error {
	zap.L().Info("Preparing resolver image context...")

	buildContext := r.generateBuildContextPath()
	if err := os.MkdirAll(buildContext, os.ModePerm); err != nil {
		return fmt.Errorf("creating build context dir %s: %w", buildContext, err)
	}

	if localPackagesPath != "" {
		dest := r.generateRPMPathInBuildContext()
		if err := os.MkdirAll(dest, os.ModePerm); err != nil {
			return fmt.Errorf("creating rpm directory in resolver dir: %w", err)
		}

		rpmNames, err := rpm.CopyRPMs(localPackagesPath, dest)
		if err != nil {
			return fmt.Errorf("copying local rpms to %s: %w", dest, err)
		}
		r.rpms = rpmNames
	}

	if err := r.writeDockerfile(localPackagesPath, packages); err != nil {
		return fmt.Errorf("writing dockerfile: %w", err)
	}

	zap.L().Info("Resolver image context setup successful")
	return nil
}

func (r *Resolver) writeDockerfile(localPackagesPath string, packages *image.Packages) error {
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
		RegCode:   packages.RegCode,
		AddRepo:   r.generateAddRepoStr(packages.AdditionalRepos),
		CacheDir:  r.generateResolverImgRPMRepoPath(),
		PkgList:   strings.Join(r.generatePKGForResolve(packages), " "),
	}

	if localPackagesPath != "" {
		values.FromRPMPath = filepath.Base(r.generateRPMPathInBuildContext())
		values.ToRPMPath = r.generateResolverImgLocalRPMDirPath()
	}

	data, err := template.Parse(dockerfileName, dockerfileTemplate, &values)
	if err != nil {
		return fmt.Errorf("parsing %s template: %w", dockerfileName, err)
	}

	filename := filepath.Join(r.generateBuildContextPath(), dockerfileName)
	if err = os.WriteFile(filename, []byte(data), fileio.NonExecutablePerms); err != nil {
		return fmt.Errorf("writing prepare base image script %s: %w", filename, err)
	}

	return nil
}

func (r *Resolver) generateAddRepoStr(repos []image.AddRepo) string {
	list := []string{}
	for _, repo := range repos {
		list = append(list, repo.URL)
	}

	return strings.Join(list, " ")
}

func (r *Resolver) generatePKGForResolve(packages *image.Packages) []string {
	list := []string{}

	if len(packages.PKGList) > 0 {
		list = append(list, packages.PKGList...)
	}

	if len(r.rpms) > 0 {
		// generate RPM paths as seen in the resolver image,
		// needed so that 'zypper install' can locate the rpms
		for _, name := range r.rpms {
			list = append(list, filepath.Join(r.generateResolverImgLocalRPMDirPath(), name))
		}
	}
	return list
}

func (r *Resolver) generatePKGInstallList(packages *image.Packages) []string {
	list := []string{}

	if len(packages.PKGList) > 0 {
		list = append(list, packages.PKGList...)
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
	return filepath.Join(r.dir, "resolver-image-build")
}

// path to the rpms directory in the resolver build context, as seen in the EIB image
func (r *Resolver) generateRPMPathInBuildContext() string {
	return filepath.Join(r.generateBuildContextPath(), "rpms")
}

// path to rpm cache directory, as seen in the resolver image
func (r *Resolver) generateResolverImgRPMRepoPath() string {
	return filepath.Join(os.TempDir(), rpmRepoName)
}

// path to the directory containing local rpms, as seen in the resolver image
func (r *Resolver) generateResolverImgLocalRPMDirPath() string {
	return filepath.Join(r.generateResolverImgRPMRepoPath(), "local")
}
