#!/bin/bash
set -euo pipefail

mkdir -p /etc/venv-salt-minion/

{{ if .GetSSL }}
curl -k -o /etc/pki/trust/anchors/suma-cert.pem https://{{ .Host }}/pub/RHN-ORG-TRUSTED-SSL-CERT
update-ca-certificates
{{ end }}

cat <<EOF > /etc/venv-salt-minion/minion
master: {{ .Host }}

grains:
  susemanager:
    activation_key: "{{ .ActivationKey }}"

server_id_use_crc: adler32
enable_legacy_startup_events: False
enable_fqdns_grains: False

EOF

systemctl restart venv-salt-minion || true
