#!/bin/bash
set -euo pipefail

{{ if or ( .HttpProxy ) ( .HttpsProxy ) }}
sed -i 's|PROXY_ENABLED=.*|PROXY_ENABLED="yes"|g' /etc/sysconfig/proxy
{{ end -}}

{{ if .HttpProxy -}}
sed -i 's|HTTP_PROXY=.*|HTTP_PROXY="{{ .HttpProxy }}"|g' /etc/sysconfig/proxy
{{ end -}}

{{ if .HttpsProxy -}}
sed -i 's|HTTPS_PROXY=.*|HTTPS_PROXY="{{ .HttpsProxy }}"|g' /etc/sysconfig/proxy
{{ end -}}

{{ if .NoProxy -}}
sed -i 's|NO_PROXY=.*|NO_PROXY="{{ .NoProxy }}"|g' /etc/sysconfig/proxy
{{ end -}}

