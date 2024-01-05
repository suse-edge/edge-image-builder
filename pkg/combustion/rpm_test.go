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

func TestSkipRPMConfigurePositive(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	// no rpm dir, not pkg configured
	assert.True(t, skipRPMconfigre(ctx))

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		AddRepos: []string{"repo1"},
	}

	// additional repo defined, but no rpm dir or pkg specified
	assert.True(t, skipRPMconfigre(ctx))

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		AddRepos: []string{"repo1"},
		RegCode:  "foo.bar",
	}

	// additional repo and reg code defined, but no rpm dir or pkg specified
	assert.True(t, skipRPMconfigre(ctx))
}

func TestSkipRPMConfigureNegative(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList: []string{"pkg1", "pkg2"},
		RegCode: "foo.bar",
	}

	// pkg from PackageHub defined
	assert.False(t, skipRPMconfigre(ctx))

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList:  []string{"pkg1", "pkg2"},
		AddRepos: []string{"repo1"},
	}

	// third party pkg defined
	assert.False(t, skipRPMconfigre(ctx))

	rpmSourceDir := filepath.Join(ctx.ImageConfigDir, userRPMsDir)
	require.NoError(t, os.Mkdir(rpmSourceDir, 0o755))

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{}

	// rpm dir defined with standalone rpms
	assert.False(t, skipRPMconfigre(ctx))

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		AddRepos: []string{"repo1"},
	}

	// rpm dir defined with rpms that require third party repositories
	assert.False(t, skipRPMconfigre(ctx))
}

func TestIsResolutionNeeded(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList: []string{"pkg1", "pkg2"},
		RegCode: "foo.bar",
	}

	// pkg from PackageHub resolution needed
	assert.True(t, isResolutionNeeded(ctx))

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		PKGList:  []string{"pkg1", "pkg2"},
		AddRepos: []string{"repo1"},
	}

	// pkg from a third party repo resolution needed
	assert.True(t, isResolutionNeeded(ctx))

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{
		AddRepos: []string{"repo1"},
	}

	// an rpm from a third party repository resolution needed
	assert.True(t, isResolutionNeeded(ctx))

	ctx.ImageDefinition.OperatingSystem.Packages = image.Packages{}

	// standalone rpm that does not need resolution
	assert.False(t, isResolutionNeeded(ctx))
}

func TestConfigureRPMs(t *testing.T) {
	ctx, teardown := setupContext(t)
	defer teardown()

	// no pkg defined no RPM dir
	scripts, err := configureRPMs(ctx)

	require.NoError(t, err)
	assert.Nil(t, scripts)

	rpmSourceDir := filepath.Join(ctx.ImageConfigDir, userRPMsDir)
	require.NoError(t, os.Mkdir(rpmSourceDir, 0o755))

	file1, err := os.Create(filepath.Join(rpmSourceDir, "rpm1.rpm"))
	require.NoError(t, err)

	file2, err := os.Create(filepath.Join(rpmSourceDir, "rpm2.rpm"))
	require.NoError(t, err)

	defer file1.Close()
	defer file2.Close()

	// standalone RPM in dir
	scripts, err = configureRPMs(ctx)

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
