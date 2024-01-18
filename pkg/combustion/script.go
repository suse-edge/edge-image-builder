package combustion

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/template"
)

//go:embed templates/script-base.sh.tpl
var combustionScriptBase string

func assembleScript(scripts []string, networkScript string) (string, error) {
	b := new(strings.Builder)

	values := struct {
		NetworkScript string
	}{
		NetworkScript: networkScript,
	}

	data, err := template.Parse("combustion-base", combustionScriptBase, values)
	if err != nil {
		return "", fmt.Errorf("parsing combustion base template: %w", err)
	}

	_, err = b.WriteString(data)
	if err != nil {
		return "", fmt.Errorf("writing script base: %w", err)
	}

	// Use alphabetical ordering for determinism
	slices.Sort(scripts)

	// Add a call to each script that was added to the combustion directory
	for _, filename := range scripts {
		_, err = b.WriteString(scriptExecutor(filename))
		if err != nil {
			return "", fmt.Errorf("appending script %s: %w", filename, err)
		}
	}

	return b.String(), nil
}

func scriptExecutor(name string) string {
	return fmt.Sprintf("./%s\n", name)
}
