package combustion

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"go.uber.org/zap"
)

// configureComponent defines the combustion component contract.
// Each component (e.g. "users") receives the necessary dir structure and
// additional values it should be operating with through a Context object.
//
// configureComponent returns a slice of scripts which should be executed as part of the Combustion script.
// Result can also be an empty slice or nil if this is not necessary.
type configureComponent func(context *image.Context) ([]string, error)

// Configure iterates over all separate Combustion components and configures them independently.
// If all of those are successful, the Combustion script is assembled and written to the file system.
func Configure(ctx *image.Context) error {
	var combustionScripts []string

	// EIB Combustion script prefix ranges:
	// 00-09 -- Networking
	// 10-19 -- Operating System
	// 20-24 -- Kubernetes
	// 25-29 -- User Workloads
	// 30-39 -- SUSE Product Integration
	// 40-49 -- Miscellaneous

	// Component order rationale:
	// - Message has no effect on the system, so this can go anywhere
	// - Custom scripts should be early to allow the most flexibility in the user
	//   being able to override/preempt the built-in behavior
	// - Elemental & SUMA must come after RPMs since the user must provide the
	//   elemental and venv-salt-minion RPMs manually
	type componentWrapper struct {
		name     string
		runnable configureComponent
	}
	combustionComponents := []componentWrapper{
		{
			name:     messageComponentName,
			runnable: configureMessage,
		},
		{
			name:     customComponentName,
			runnable: configureCustomFiles,
		},
		{
			name:     timeComponentName,
			runnable: configureTime,
		},
		{
			name:     timesyncComponentName,
			runnable: configureTimesync,
		},
		{
			name:     networkComponentName,
			runnable: configureNetwork,
		},
		{
			name:     usersComponentName,
			runnable: configureUsers,
		},
		{
			name:     proxyComponentName,
			runnable: configureProxy,
		},
		{
			name:     rpmComponentName,
			runnable: configureRPMs,
		},
		{
			name:     systemdComponentName,
			runnable: configureSystemd,
		},
		{
			name:     elementalComponentName,
			runnable: configureElemental,
		},
		{
			name:     sumaComponentName,
			runnable: configureSuma,
		},
		{
			name:     registryComponentName,
			runnable: configureRegistry,
		},
		{
			name:     keymapComponentName,
			runnable: configureKeymap,
		},
		{
			name:     k8sComponentName,
			runnable: configureKubernetes,
		},
		{
			name:     certsComponentName,
			runnable: configureCertificates,
		},
	}

	for _, component := range combustionComponents {
		scripts, err := component.runnable(ctx)
		if err != nil {
			return fmt.Errorf("configuring component %q: %w", component.name, err)
		}

		combustionScripts = append(combustionScripts, scripts...)
	}

	var networkScript string
	if isComponentConfigured(ctx, networkConfigDir) {
		networkScript = networkConfigScriptName
	}

	script, err := assembleScript(combustionScripts, networkScript)
	if err != nil {
		return fmt.Errorf("assembling script: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, "script")
	if err = os.WriteFile(filename, []byte(script), fileio.ExecutablePerms); err != nil {
		return fmt.Errorf("writing script: %w", err)
	}

	return nil
}

func generateComponentPath(ctx *image.Context, componentDir string) string {
	return filepath.Join(ctx.ImageConfigDir, componentDir)
}

func isComponentConfigured(ctx *image.Context, componentDir string) bool {
	if componentDir == "" {
		zap.S().Warn("Component dir not provided")
		return false
	}

	componentPath := generateComponentPath(ctx, componentDir)

	_, err := os.Stat(componentPath)
	if err == nil {
		return true
	}

	if !errors.Is(err, fs.ErrNotExist) {
		zap.S().Warnf("Searching for component directory (%s) failed, component will be skipped: %s",
			componentDir, err)
	}

	return false
}
