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

func TestSkipRPMComponent_InvalidDefinition(t *testing.T) {
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

func TestSkipRPMComponent_PopulatedPackageList(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList: []string{"pkg1", "pkg2"},
	}

	assert.False(t, SkipRPMComponent(ctx))
}

func TestSkipRPMComponent_EmptyRPMDir(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmDir := filepath.Join(ctx.ImageConfigDir, rpmDir)
	require.NoError(t, os.Mkdir(rpmDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(rpmDir))
	}()

	assert.True(t, SkipRPMComponent(ctx))
}

func TestSkipRPMComponent_FullConfig(t *testing.T) {
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

func TestConfigureRPMs_Skipped(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	var c Combustion

	scripts, err := c.configureRPMs(ctx)

	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureRPMs_ResolutionFailures(t *testing.T) {
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
			c := Combustion{
				RPMResolver:    test.rpmResolver,
				RPMRepoCreator: test.rpmRepoCreator,
			}

			_, err := c.configureRPMs(ctx)
			require.Error(t, err)
			assert.EqualError(t, err, test.expectedErr)
		})
	}
}

func TestConfigureRPMs_GPGFailures(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmDir := filepath.Join(ctx.ImageConfigDir, rpmDir)
	require.NoError(t, os.Mkdir(rpmDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(rpmDir))
	}()

	require.NoError(t, os.WriteFile(filepath.Join(rpmDir, "test.rpm"), nil, 0o600))

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
			expectedErr: "fetching local RPM config: found existing 'gpg-keys' directory, but GPG validation is disabled",
		},
		{
			name:         "Enabled GPG validation, but empty GPG dir",
			createGPGDir: true,
			expectedErr:  "fetching local RPM config: 'gpg-keys' directory exists but it is empty",
		},
		{
			name:        "Enabled GPG validation, but missing GPG dir",
			pkgs:        image.Packages{},
			expectedErr: "fetching local RPM config: GPG validation is enabled, but 'gpg-keys' directory is missing for side-loaded RPMs",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.ImageDefinition.OperatingSystem.Packages = test.pkgs

			gpgDir := filepath.Join(rpmDir, gpgDir)
			if test.createGPGDir {
				require.NoError(t, os.Mkdir(gpgDir, 0o755))
			}

			var c Combustion

			_, err := c.configureRPMs(ctx)
			require.Error(t, err)
			assert.EqualError(t, err, test.expectedErr)

			require.NoError(t, os.RemoveAll(gpgDir))
		})
	}
}

func TestConfigureRPMs_SuccessfulConfig(t *testing.T) {
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

	require.NoError(t, os.WriteFile(filepath.Join(gpgDir, "some-key"), nil, 0o600))

	c := Combustion{
		RPMRepoCreator: mockRPMRepoCreator{
			createFunc: func(path string) error {
				return nil
			},
		},
		RPMResolver: mockRPMResolver{
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
		},
	}

	scripts, err := c.configureRPMs(ctx)
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
