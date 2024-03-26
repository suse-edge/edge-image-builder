package combustion

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

type mockRPMResolver struct {
	resolveFunc func(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (rpmDir string, pkgList []string, err error)
}

func (m mockRPMResolver) Resolve(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (rpmDir string, pkgList []string, err error) {
	if m.resolveFunc != nil {
		return m.resolveFunc(packages, localRPMConfig, outputDir)
	}

	panic("not implemented")
}

type mockRPMRepoCreator struct {
	createFunc func(path string) error
}

func (mr mockRPMRepoCreator) Create(path string) error {
	if mr.createFunc != nil {
		return mr.createFunc(path)
	}

	panic("not implemented")
}

func TestSkipRPMComponentTrue(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	tests := []struct {
		name     string
		packages image.Packages
	}{
		{
			name: "No RPM directory or package list",
		},
		{
			name: "Additional repository without an RPM directory or package list",
			packages: image.Packages{
				AdditionalRepos: []image.AddRepo{
					{
						URL: "https://foo.bar",
					},
				},
			},
		},
		{
			name: "Additional repository and registration code without RPM directory or package list",
			packages: image.Packages{
				AdditionalRepos: []image.AddRepo{
					{
						URL: "https://foo.bar",
					},
				},
				RegCode: "foo.bar",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.ImageDefinition.OperatingSystem.Packages = test.packages
			assert.True(t, SkipRPMComponent(ctx))
		})
	}

}

func TestSkipRPMComponentProvidedPKGList(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList: []string{"pkg1", "pkg2"},
	}

	assert.False(t, SkipRPMComponent(ctx))
}

func TestSkipRPMComponentRPMDirNoRPMs(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmDir := filepath.Join(ctx.ImageConfigDir, rpmDir)
	require.NoError(t, os.Mkdir(rpmDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(rpmDir))
	}()

	assert.True(t, SkipRPMComponent(ctx))
}

func TestSkipRPMComponentFullConfig(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmDir := filepath.Join(ctx.ImageConfigDir, rpmDir)
	require.NoError(t, os.Mkdir(rpmDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(rpmDir))
	}()

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList: []string{"pkg1", "pkg2"},
		AdditionalRepos: []image.AddRepo{
			{
				URL: "https://foo.bar",
			},
		},
		RegCode: "foo.bar",
	}

	assert.False(t, SkipRPMComponent(ctx))
}

func TestConfigureRPMSSkipComponent(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	scripts, err := configureRPMs(ctx)

	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureRPMSError(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	// do not skip RPM component
	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList: []string{"foo", "bar"},
		AdditionalRepos: []image.AddRepo{
			{
				URL: "https://foo.bar",
			},
		},
	}

	tests := []struct {
		name           string
		rpmResolver    mockRPMResolver
		rpmRepoCreator mockRPMRepoCreator
		expectedErr    string
	}{
		{
			name: "Resolving RPM dependencies fails",
			rpmResolver: mockRPMResolver{
				resolveFunc: func(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (rpmDir string, pkgList []string, err error) {
					return "", nil, fmt.Errorf("resolution failed")
				},
			},
			expectedErr: "resolving rpm/package dependencies: resolution failed",
		},
		{
			name: "Creating RPM repository fails",
			rpmResolver: mockRPMResolver{
				resolveFunc: func(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (rpmDir string, pkgList []string, err error) {
					return "rpm-repo", []string{"foo", "bar"}, nil
				},
			},
			rpmRepoCreator: mockRPMRepoCreator{
				createFunc: func(path string) error {
					return fmt.Errorf("rpm repo creation failed")
				},
			},
			expectedErr: "creating resolved rpm repository: rpm repo creation failed",
		},
		{
			name: "Writing RPM script with empty package list",
			rpmResolver: mockRPMResolver{
				resolveFunc: func(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (rpmDir string, pkgList []string, err error) {
					return "rpm-repo", []string{}, nil
				},
			},
			rpmRepoCreator: mockRPMRepoCreator{
				createFunc: func(path string) error {
					return nil
				},
			},
			expectedErr: "writing the RPM install script 10-rpm-install.sh: package list cannot be empty",
		},
		{
			name: "Writing RPM script with empty repo path",
			rpmResolver: mockRPMResolver{
				resolveFunc: func(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (rpmDir string, pkgList []string, err error) {
					return "", []string{"foo", "bar"}, nil
				},
			},
			rpmRepoCreator: mockRPMRepoCreator{
				createFunc: func(path string) error {
					return nil
				},
			},
			expectedErr: "writing the RPM install script 10-rpm-install.sh: path to RPM repository cannot be empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.RPMResolver = test.rpmResolver
			ctx.RPMRepoCreator = test.rpmRepoCreator

			_, err := configureRPMs(ctx)
			require.Error(t, err)
			assert.EqualError(t, err, test.expectedErr)
		})
	}
}

func TestConfigureRPMSGPGDirError(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmDir := filepath.Join(ctx.ImageConfigDir, rpmDir)
	require.NoError(t, os.Mkdir(rpmDir, 0o755))
	_, err := os.Create(filepath.Join(rpmDir, "test.rpm"))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, os.RemoveAll(rpmDir))
	}()

	tests := []struct {
		name         string
		expectedErr  string
		pkgs         image.Packages
		createGPGDir bool
	}{
		{
			name:         "Disabled GPG validation, but existing GPG dir",
			createGPGDir: true,
			pkgs: image.Packages{
				NoGPGCheck: true,
			},
			expectedErr: fmt.Sprintf("found existing '%s' directory, but GPG validation is disabled", gpgDir),
		},
		{
			name:        "Enabled GPG validation, but missing GPG dir",
			pkgs:        image.Packages{},
			expectedErr: "GPG validation is enabled, but 'gpg-keys' directory is missing for side-loaded RPMs",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.ImageDefinition.OperatingSystem.Packages = test.pkgs

			gpgDir := filepath.Join(rpmDir, gpgDir)
			if test.createGPGDir {
				require.NoError(t, os.Mkdir(gpgDir, 0o755))
			}

			_, err := configureRPMs(ctx)
			require.Error(t, err)
			assert.EqualError(t, err, test.expectedErr)

			require.NoError(t, os.RemoveAll(gpgDir))
		})
	}
}

func TestConfigureRPMSSuccessfulConfig(t *testing.T) {
	expectedRepoName := "bar"
	expectedDir := "/foo/bar"
	expectedPkg := []string{"foo", "bar"}

	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList: []string{"foo", "bar"},
		AdditionalRepos: []image.AddRepo{
			{
				URL: "https://foo.bar",
			},
		},
	}

	rpmDir := filepath.Join(ctx.ImageConfigDir, rpmDir)
	require.NoError(t, os.Mkdir(rpmDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(rpmDir))
	}()

	gpgDir := filepath.Join(rpmDir, gpgDir)
	require.NoError(t, os.Mkdir(gpgDir, 0o755))

	ctx.RPMRepoCreator = mockRPMRepoCreator{
		createFunc: func(path string) error {
			return nil
		},
	}

	ctx.RPMResolver = mockRPMResolver{
		resolveFunc: func(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (string, []string, error) {
			if localRPMConfig == nil {
				return "", nil, fmt.Errorf("local rpm config is nil")
			}
			if rpmDir != localRPMConfig.RPMPath {
				return "", nil, fmt.Errorf("rpm path mismatch. Expected %s, got %s", rpmDir, localRPMConfig.RPMPath)
			}
			if gpgDir != localRPMConfig.GPGKeysPath {
				return "", nil, fmt.Errorf("gpg path mismatch. Expected %s, got %s", gpgDir, localRPMConfig.GPGKeysPath)
			}

			return expectedDir, expectedPkg, nil
		},
	}

	scripts, err := configureRPMs(ctx)
	require.NoError(t, err)
	require.NotNil(t, scripts)
	require.Len(t, scripts, 1)
	assert.Equal(t, installRPMsScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, installRPMsScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)
	zypperAR := fmt.Sprintf("zypper ar file://$ARTEFACTS_DIR/rpms/%[1]s %[1]s", expectedRepoName)
	zypperInstall := fmt.Sprintf("zypper --no-gpg-checks install -r %s -y --force-resolution --auto-agree-with-licenses %s", expectedRepoName, strings.Join(expectedPkg, " "))
	zypperRR := fmt.Sprintf("zypper rr %s", expectedRepoName)
	assert.Contains(t, foundContents, zypperAR)
	assert.Contains(t, foundContents, zypperInstall)
	assert.Contains(t, foundContents, zypperRR)
}
