#!/bin/bash
set -euo pipefail

{{ if .Timezone -}}
ln -sf /usr/share/zoneinfo/{{ .Timezone }} /etc/localtime
{{ end -}}

{{ if or (gt (len .ChronyPools) 0) (gt (len .ChronyServers) 0) }}
rm -f /etc/chrony.d/pool.conf
{{ end -}}

{{ range .ChronyPools -}}
echo "pool {{ . }} iburst" >> /etc/chrony.d/eib-sources.conf
{{ end -}}

{{ range .ChronyServers -}}
echo "server {{ . }} iburst" >> /etc/chrony.d/eib-sources.conf
{{ end -}}
