#!/bin/bash
set -euo pipefail

{{ if or ( .HTTPProxy ) ( .HTTPSProxy ) }}
sed -i 's|PROXY_ENABLED=.*|PROXY_ENABLED="yes"|g' /etc/sysconfig/proxy
{{ end -}}

{{ if .HTTPProxy -}}
sed -i 's|HTTP_PROXY=.*|HTTP_PROXY="{{ .HTTPProxy }}"|g' /etc/sysconfig/proxy
{{ end -}}

{{ if .HTTPSProxy -}}
sed -i 's|HTTPS_PROXY=.*|HTTPS_PROXY="{{ .HTTPSProxy }}"|g' /etc/sysconfig/proxy
{{ end -}}

{{ if .NoProxy -}}
sed -i 's|NO_PROXY=.*|NO_PROXY="{{ .NoProxy }}"|g' /etc/sysconfig/proxy
{{ end -}}

