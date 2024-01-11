package combustion

import (
	"bytes"
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

func TestWriteRPMScriptWithRPMRepo(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	repoName := "foo"
	pkgList := []string{"pkg1", "pkg2", "pkg3"}
	script, err := writeRPMScript(ctx, repoName, pkgList)
	require.NoError(t, err)
	assert.Equal(t, modifyRPMScriptName, script)

	expectedFilename := filepath.Join(ctx.CombustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)
	zypperAR := fmt.Sprintf("zypper ar file://%s %s", filepath.Join(combustionBasePath, repoName), repoName)
	zypperInstall := fmt.Sprintf("zypper --no-gpg-checks install -r %s -y --force-resolution --auto-agree-with-licenses %s", repoName, strings.Join(pkgList, " "))
	zypperRR := fmt.Sprintf("zypper rr %s", repoName)
	assert.Contains(t, foundContents, zypperAR)
	assert.Contains(t, foundContents, zypperInstall)
	assert.Contains(t, foundContents, zypperRR)
}

func TestWriteRPMScriptStandaloneRPM(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	repoName := ""
	pkgList := []string{"pkg1", "pkg2", "pkg3"}
	script, err := writeRPMScript(ctx, repoName, pkgList)
	require.NoError(t, err)
	assert.Equal(t, modifyRPMScriptName, script)

	expectedFilename := filepath.Join(ctx.CombustionDir, modifyRPMScriptName)
	foundBytes, err := os.ReadFile(expectedFilename)
	require.NoError(t, err)

	stats, err := os.Stat(expectedFilename)
	require.NoError(t, err)
	assert.Equal(t, fileio.ExecutablePerms, stats.Mode())

	foundContents := string(foundBytes)
	zypperAR := "zypper ar file:/"
	zypperInstall := fmt.Sprintf("zypper --no-gpg-checks install -y --force-resolution --auto-agree-with-licenses %s", strings.Join(pkgList, " "))
	zypperRR := "zypper rr"
	assert.Contains(t, foundContents, zypperInstall)
	assert.NotContains(t, foundContents, zypperAR)
	assert.NotContains(t, foundContents, zypperRR)
}

func TestWriteRPMScriptEmptyPKGList(t *testing.T) {
	_, err := writeRPMScript(nil, "", []string{})
	require.Error(t, err)
	require.ErrorContains(t, err, "package list cannot be empty")
}

func TestSkipRPMConfigure(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmDir := filepath.Join(ctx.ImageConfigDir, userRPMsDir)

	tests := []struct {
		name           string
		packages       image.Packages
		needsRPMDir    bool
		expectedResult bool
	}{
		{
			name:           "No RPM directory or package list",
			expectedResult: true,
		},
		{
			name: "Additional repository without an RPM directory or package list",
			packages: image.Packages{
				AdditionalRepos: []string{"https://foo.bar"},
			},
			expectedResult: true,
		},
		{
			name: "Additional repository and registration code without RPM directory or package list",
			packages: image.Packages{
				AdditionalRepos: []string{"https://foo.bar"},
				RegCode:         "foo.bar",
			},
			expectedResult: true,
		},
		{
			name: "Package list provided",
			packages: image.Packages{
				PKGList: []string{"pkg1", "pkg2"},
			},
			expectedResult: false,
		},
		{
			name:           "RPM provided in RPM dir",
			needsRPMDir:    true,
			expectedResult: false,
		},
		{
			name:        "Full configuration",
			needsRPMDir: true,
			packages: image.Packages{
				PKGList:         []string{"pkg1", "pkg2"},
				AdditionalRepos: []string{"https://foo.bar"},
				RegCode:         "foo.bar",
			},
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.ImageDefinition.OperatingSystem.Packages = test.packages

			if test.expectedResult {
				assert.True(t, skipRPMComponent(ctx))
			} else {
				if test.needsRPMDir && !isComponentConfigured(ctx, userRPMsDir) {
					require.NoError(t, os.Mkdir(rpmDir, 0o755))
				} else if !test.needsRPMDir {
					// remove RPM dir if test does not need it
					// in order to ensure correct testing environment
					require.NoError(t, os.RemoveAll(rpmDir))
				}

				assert.False(t, skipRPMComponent(ctx))
			}
		})
	}
}

func TestIsResolutionNeeded(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	tests := []struct {
		name           string
		packages       image.Packages
		expectedResult bool
	}{
		{
			name: "RPM/package resolution from PackageHub",
			packages: image.Packages{
				RegCode: "foo.bar",
			},
			expectedResult: true,
		},
		{
			name: "RPM/package resolution from third party repository",
			packages: image.Packages{
				AdditionalRepos: []string{"https://foo.bar"},
			},
			expectedResult: true,
		},
		{
			name: "RPM/package resolution PackageHub and third party repository",
			packages: image.Packages{
				AdditionalRepos: []string{"https://foo.bar"},
				RegCode:         "foo.bar",
			},
			expectedResult: true,
		},
		{
			name:           "Standalone RPM",
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.ImageDefinition.OperatingSystem.Packages = test.packages

			if test.expectedResult {
				assert.True(t, isResolutionNeeded(ctx))
			} else {
				assert.False(t, isResolutionNeeded(ctx))
			}
		})
	}
}

func TestConfigureRPMSSkipComponent(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	scripts, err := configureRPMs(ctx)

	require.NoError(t, err)
	assert.Nil(t, scripts)
}

func TestConfigureRPMSStandaloneRPM(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	rpmSourceDir := filepath.Join(ctx.ImageConfigDir, userRPMsDir)
	require.NoError(t, os.Mkdir(rpmSourceDir, 0o755))

	file1, err := os.Create(filepath.Join(rpmSourceDir, "rpm1.rpm"))
	require.NoError(t, err)

	file2, err := os.Create(filepath.Join(rpmSourceDir, "rpm2.rpm"))
	require.NoError(t, err)

	defer file1.Close()
	defer file2.Close()

	scripts, err := configureRPMs(ctx)

	require.NoError(t, err)
	require.NotNil(t, scripts)
	assert.Equal(t, modifyRPMScriptName, scripts[0])

	_, err = os.Stat(filepath.Join(ctx.CombustionDir, "rpm1.rpm"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(ctx.CombustionDir, "rpm2.rpm"))
	require.NoError(t, err)

	expectedFilename := filepath.Join(ctx.CombustionDir, modifyRPMScriptName)
	_, err = os.ReadFile(expectedFilename)
	require.NoError(t, err)
}

func TestPrepareRepoCommand(t *testing.T) {
	const (
		testPath = "/foo/bar"
	)

	cmd := prepareRepoCommand(testPath, &bytes.Buffer{})

	assert.Equal(t, cmd.Path, createRepoExec)
	require.Len(t, cmd.Args, 2)
	assert.Equal(t, cmd.Args[1], testPath)
	require.NotNil(t, cmd.Stderr)
	require.NotNil(t, cmd.Stdout)
}
