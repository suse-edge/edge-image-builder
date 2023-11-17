package build

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/template"
)

//go:embed scripts/grub/guestfish-snippet.tpl
var guestfishSnippet string

func (b *Builder) generateGRUBGuestfishCommands() (string, error) {
	// Nothing to do if there aren't any args. Return an empty string that will be injected
	// into the raw image guestfish modification, effectively doing nothing but not breaking
	// the guestfish command
	if b.imageConfig.OperatingSystem.KernelArgs == nil {
		return "", nil
	}

	argLine := strings.Join(b.imageConfig.OperatingSystem.KernelArgs, " ")
	values := struct {
		KernelArgs string
	}{
		KernelArgs: argLine,
	}

	snippet, err := template.Parse("guestfish-snippet", guestfishSnippet, values)
	if err != nil {
		return "", fmt.Errorf("parsing guestfish template: %w", err)
	}

	return snippet, nil
}
