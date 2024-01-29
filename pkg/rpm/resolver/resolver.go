package resolver

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"go.uber.org/zap"
)

const (
	resolverImageRef = "pkg-resolver"
	dockerfileName   = "Dockerfile"
	rpmRepoName      = "rpm-repo"
	gpgDirName       = "gpg-keys"
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
	// helper property, contains RPM paths that will be used for resolution in the
	// resolver image
	rpmPaths []string
	// helper property, contains the paths to the gpgKeys that will be used to validate
	// the RPM signatures in the resolver image
	gpgKeyPaths []string
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
// - localRPMConfig - configuration for locally provided RPMs
//
// - outputDir - directory in which the resolver will create a directory containing the resolved rpms.
func (r *Resolver) Resolve(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (rpmDirPath string, pkgList []string, err error) {
	zap.L().Info("Resolving package dependencies...")

	if err = r.buildBase(); err != nil {
		return "", nil, fmt.Errorf("building base resolver image: %w", err)
	}

	if err = r.prepare(localRPMConfig, packages); err != nil {
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

func (r *Resolver) prepare(localRPMConfig *image.LocalRPMConfig, packages *image.Packages) error {
	zap.L().Info("Preparing resolver image context...")

	buildContext := r.generateBuildContextPath()
	if err := os.MkdirAll(buildContext, os.ModePerm); err != nil {
		return fmt.Errorf("creating build context dir %s: %w", buildContext, err)
	}

	if localRPMConfig != nil {
		if err := r.prepareLocalRPMs(localRPMConfig); err != nil {
			return fmt.Errorf("preparing local RPMs for resolver image build: %w", err)
		}
	}

	if err := r.writeDockerfile(localPackagesPath, packages); err != nil {
		return fmt.Errorf("writing dockerfile: %w", err)
	}

	zap.L().Info("Resolver image context setup successful")
	return nil
}

func (r *Resolver) prepareLocalRPMs(localRPMConfig *image.LocalRPMConfig) error {
	rpmDest := r.generateRPMPathInBuildContext()
	if err := fileio.CopyFiles(localRPMConfig.RPMPath, rpmDest, ".rpm", false); err != nil {
		return fmt.Errorf("copying local rpms to %s: %w", rpmDest, err)
	}

	rpmPaths, err := r.generateResolverImgRPMPaths()
	if err != nil {
		return fmt.Errorf("constructing list of rpm paths that need to be installed: %w", err)
	}
	// path to rpms as seen in the resolver image
	r.rpmPaths = rpmPaths

	if localRPMConfig.GPGKeysPath != "" {
		gpgDest := r.generateGPGPathInBuildContext()
		if err := fileio.CopyFiles(localRPMConfig.GPGKeysPath, gpgDest, "", false); err != nil {
			return fmt.Errorf("copying local GPG keys to %s: %w", gpgDest, err)
		}

		gpgPaths, err := r.generateResolverImgGPGPaths()
		if err != nil {
			return fmt.Errorf("constructing list of gpg paths that need to be imported: %w", err)
		}

		// path to GPG keys as seen in the resolver image
		r.gpgKeyPaths = gpgPaths
	}

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

	if len(r.rpmPaths) > 0 {
		// generate RPM paths as seen in the resolver image,
		// needed so that 'zypper install' can locate the rpms
		for _, name := range r.rpmPaths {
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

	if len(r.rpmPaths) > 0 {
		// generate the RPMs as package names,
		// so that zypper can locate them in the RPM repository
		for _, path := range r.rpmPaths {
			list = append(list, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
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

// path to the gpg keys directory in the resolver build context, as seen in the EIB image
func (r *Resolver) generateGPGPathInBuildContext() string {
	return filepath.Join(r.generateBuildContextPath(), gpgDirName)
}

func (r *Resolver) generateResolverImgGPGPaths() (gpgPathList []string, err error) {
	gpgs, err := os.ReadDir(r.generateGPGPathInBuildContext())
	if err != nil {
		return nil, fmt.Errorf("reading GPG source dir: %w", err)
	}

	for _, gpg := range gpgs {
		gpgPathList = append(gpgPathList, filepath.Join(r.generateResolverImgGPGKeysPath(), gpg.Name()))
	}

	return gpgPathList, nil
}

func (r *Resolver) generateResolverImgRPMPaths() (rpmPathList []string, err error) {
	rpms, err := os.ReadDir(r.generateRPMPathInBuildContext())
	if err != nil {
		return nil, fmt.Errorf("reading RPM source dir: %w", err)
	}

	for _, rpm := range rpms {
		rpmPathList = append(rpmPathList, filepath.Join(r.generateResolverImgLocalRPMDirPath(), rpm.Name()))
	}

	return rpmPathList, nil
}

// path to the GPG keys directory, as seen in the resolver image
func (r *Resolver) generateResolverImgGPGKeysPath() string {
	return filepath.Join(os.TempDir(), gpgDirName)
}

// path to rpm cache directory, as seen in the resolver image
func (r *Resolver) generateResolverImgRPMRepoPath() string {
	return filepath.Join(os.TempDir(), rpmRepoName)
}

// path to the directory containing local rpms, as seen in the resolver image
func (r *Resolver) generateResolverImgLocalRPMDirPath() string {
	return filepath.Join(r.generateResolverImgRPMRepoPath(), "local")
}
