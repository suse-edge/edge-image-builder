package build

import (
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/config"
)

const (
	messageScriptName = "message.sh"
)

func ConfigureMessage(buildConfig *config.BuildConfig) error {
	// There's currently intentionally no way to disable this in the image config,
	// but we may want to add that in the future
	err := copyCombustionFile(filepath.Join("scripts", "message", messageScriptName), buildConfig)
	if err != nil {
		return err
	}

	buildConfig.AddCombustionScript(messageScriptName)
	return nil
}
