package combustion

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	"github.com/suse-edge/edge-image-builder/pkg/log"
	"github.com/suse-edge/edge-image-builder/pkg/template"
	"github.com/suse-edge/edge-image-builder/pkg/version"
)

const (
	messageScriptName    = "48-message.sh"
	messageComponentName = "identifier"
)

//go:embed templates/48-message.sh.tpl
var messageScript string

func configureMessage(ctx *image.Context) ([]string, error) {
	values := struct {
		Version string
	}{
		Version: version.GetVersion(),
	}

	data, err := template.Parse(messageScriptName, messageScript, &values)
	if err != nil {
		return nil, fmt.Errorf("parsing message script template: %w", err)
	}

	filename := filepath.Join(ctx.CombustionDir, messageScriptName)
	err = os.WriteFile(filename, []byte(data), fileio.ExecutablePerms)
	if err != nil {
		log.AuditComponentFailed(messageComponentName)
		return nil, fmt.Errorf("writing message script: %w", err)
	}

	log.AuditComponentSuccessful(messageComponentName)
	return []string{messageScriptName}, nil
}
