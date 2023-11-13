package build

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
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

	tmpl, err := template.New("guestfish-snippet").Parse(guestfishSnippet)
	if err != nil {
		return "", fmt.Errorf("building template for GRUB guestfish snippet: %w", err)
	}

	argLine := strings.Join(b.imageConfig.OperatingSystem.KernelArgs, " ")
	values := struct {
		KernelArgs string
	}{
		KernelArgs: argLine,
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, values)
	if err != nil {
		return "", fmt.Errorf("applying GRUB guestfish snippet: %w", err)
	}

	snippet := buff.String()
	return snippet, nil
}
