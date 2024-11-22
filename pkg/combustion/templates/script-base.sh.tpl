#!/bin/bash
set -euo pipefail

{{ if .NetworkScript -}}
# combustion: prepare network

if [ "${1-}" = "--prepare" ]; then
    ./{{ .NetworkScript }}
    exit 0
fi
{{- else -}}
# combustion: network
{{- end }}

# Redirect output to the console
exec > >(exec tee -a /dev/tty0) 2>&1

cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1

{{ if eq .ImageType "combustion" -}}
export ARTEFACTS_DIR=../artefacts
{{- else -}}
mount -o ro /dev/disk/by-label/INSTALL /mnt
export ARTEFACTS_DIR=/mnt/artefacts
{{- end }}

{{ range .Scripts -}}
echo "Running {{ . }}"
./{{ . }}

{{ end -}}

{{ if ne .ImageType "combustion" -}}
umount /mnt
{{- end }}
