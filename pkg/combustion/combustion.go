package combustion

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

//go:embed scripts/script_base.sh
var combustionScriptBase string

func GenerateScript(scripts []string) (string, error) {
	b := new(strings.Builder)

	_, err := b.WriteString(combustionScriptBase)
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
