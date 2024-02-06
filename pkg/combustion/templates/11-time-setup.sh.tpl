#!/bin/bash
set -euo pipefail

{{ if .Timezone -}}
ln -sf /usr/share/zoneinfo/{{ .Timezone }} /etc/localtime
{{ end -}}

{{ if or (gt (len .Pools) 0) (gt (len .Servers) 0) }}
rm -f /etc/chrony.d/pool.conf
{{ end -}}

{{ range .Pools -}}
echo "pool {{ . }} iburst" >> /etc/chrony.d/eib-sources.conf
{{ end -}}

{{ range .Servers -}}
echo "server {{ . }} iburst" >> /etc/chrony.d/eib-sources.conf
{{ end -}}
