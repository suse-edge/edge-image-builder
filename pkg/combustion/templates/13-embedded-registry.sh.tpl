#!/bin/bash
set -euo pipefail

mkdir /opt/hauler

mv {{ .EmbeddedRegistryTar }} /opt/hauler/{{ .EmbeddedRegistryTar }}
mv hauler /usr/bin/hauler

cat <<- EOF > /etc/systemd/system/eib-embedded-registry.service
  [Unit]
  Description=Load and Serve Embedded Registry
  After=network.target

  [Service]
  Type=simple
  User=root
  WorkingDirectory=/opt/hauler
  ExecStartPre=/usr/bin/hauler store load {{ .EmbeddedRegistryTar }}
  ExecStart=/usr/bin/hauler store serve
  Restart=on-failure

  [Install]
  WantedBy=multi-user.target
EOF

systemctl enable eib-embedded-registry.service