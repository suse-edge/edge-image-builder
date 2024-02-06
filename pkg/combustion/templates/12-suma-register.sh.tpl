#!/bin/bash
set -euo pipefail

mkdir -p /etc/venv-salt-minion/

cat <<EOF > /etc/venv-salt-minion/minion
master: {{ .Host }}

grains:
  susemanager:
    activation_key: "{{ .ActivationKey }}"

server_id_use_crc: adler32
enable_legacy_startup_events: False
enable_fqdns_grains: False

EOF

systemctl enable venv-salt-minion || true
