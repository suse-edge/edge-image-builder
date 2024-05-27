#!/bin/bash
set -euo pipefail

mkdir -p /opt/hauler
cp {{ .RegistryDir }}/hauler /opt/hauler/hauler
cp {{ .RegistryDir }}/*-{{ .RegistryTarSuffix }} /opt/hauler/

cat <<- EOF > /etc/systemd/system/eib-embedded-registry.service
[Unit]
Description=Load and Serve Embedded Registry
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/hauler
ExecStartPre=/bin/sh -c '/opt/hauler/hauler store load *-{{ .RegistryTarSuffix }}'
ExecStart=/opt/hauler/hauler store serve registry -p {{ .RegistryPort }}
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl enable eib-embedded-registry.service
