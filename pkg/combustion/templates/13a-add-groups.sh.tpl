#!/bin/bash
set -euo pipefail

{{- range . }}
groupadd -f {{ .Name }}
{{- end }}