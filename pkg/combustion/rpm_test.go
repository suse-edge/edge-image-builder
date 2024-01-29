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

func TestSkipRPMComponentRPMInDir(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmDir := filepath.Join(ctx.ImageConfigDir, userRPMsDir)
	require.NoError(t, os.Mkdir(rpmDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(rpmDir))
	}()

	assert.False(t, SkipRPMComponent(ctx))
}

func TestSkipRPMComponentFullConfig(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmDir := filepath.Join(ctx.ImageConfigDir, userRPMsDir)
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

	rpmDir := filepath.Join(ctx.ImageConfigDir, userRPMsDir)
	require.NoError(t, os.Mkdir(rpmDir, 0o755))
	defer func() {
		require.NoError(t, os.RemoveAll(rpmDir))
	}()

	ctx.RPMRepoCreator = mockRPMRepoCreator{
		createFunc: func(path string) error {
			return nil
		},
	}

	ctx.RPMResolver = mockRPMResolver{
		resolveFunc: func(packages *image.Packages, localRPMConfig *image.LocalRPMConfig, outputDir string) (string, []string, error) {
			return expectedDir, expectedPkg, nil
		},
	}

	scripts, err := configureRPMs(ctx)
	require.NoError(t, err)
	require.NotNil(t, scripts)
	require.Len(t, scripts, 1)
	assert.Equal(t, modifyRPMScriptName, scripts[0])

	expectedFilename := filepath.Join(ctx.CombustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)
	combustionBasePath := "/dev/shm/combustion/config"
	zypperAR := fmt.Sprintf("zypper ar file://%s %s", filepath.Join(combustionBasePath, expectedRepoName), expectedRepoName)
	zypperInstall := fmt.Sprintf("zypper --no-gpg-checks install -r %s -y --force-resolution --auto-agree-with-licenses %s", expectedRepoName, strings.Join(expectedPkg, " "))
	zypperRR := fmt.Sprintf("zypper rr %s", expectedRepoName)
	assert.Contains(t, foundContents, zypperAR)
	assert.Contains(t, foundContents, zypperInstall)
	assert.Contains(t, foundContents, zypperRR)
}
