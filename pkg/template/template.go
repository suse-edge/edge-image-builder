package template

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"strings"
	"text/template"
)

var version string

func Parse(name string, contents string, templateData any) (string, error) {
	if templateData == nil {
		return "", fmt.Errorf("template data not provided")
	}

	funcs := template.FuncMap{"join": strings.Join}

	tmpl, err := template.New(name).Funcs(funcs).Parse(contents)
	if err != nil {
		return "", fmt.Errorf("parsing contents: %w", err)
	}

	var buff bytes.Buffer
	if err = tmpl.Execute(&buff, templateData); err != nil {
		return "", fmt.Errorf("applying template: %w", err)
	}

	return buff.String(), nil
}

func GetVersion() string {
	if version != "" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		fmt.Println(info)
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return fmt.Sprintf("git-%s", setting.Value)
			}
		}
	}

	return ""
}
