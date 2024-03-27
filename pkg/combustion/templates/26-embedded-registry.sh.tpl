#!/bin/bash
set -euo pipefail

mkdir /opt/hauler
cp {{ .RegistryDir }}/hauler /opt/hauler/hauler
cp {{ .RegistryDir }}/{{ .EmbeddedRegistryTar }} /opt/hauler/

cat <<- EOF > /etc/systemd/system/eib-embedded-registry.service
[Unit]
Description=Load and Serve Embedded Registry
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/hauler
ExecStartPre=/opt/hauler/hauler store load {{ .EmbeddedRegistryTar }}
ExecStart=/opt/hauler/hauler store serve registry -p {{ .RegistryPort }}
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl enable eib-embedded-registry.service