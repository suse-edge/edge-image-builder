#!/bin/bash
set -euo pipefail

{{- range . }}
{{- $gid := "" }}
{{- if (ne .GID 0 )}}
  {{- $gid = (printf "-g %v " .GID) }}
{{- end }}
groupadd -f {{ $gid }}{{ .Name }}
{{- end }}