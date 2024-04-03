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

mount -o ro /dev/disk/by-label/INSTALL /mnt
export ARTEFACTS_DIR=/mnt/artefacts

{{ range .Scripts -}}
echo "Running {{ . }}"
./{{ . }}

{{ end -}}

umount /mnt
