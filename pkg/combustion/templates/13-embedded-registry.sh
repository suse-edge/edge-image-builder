#!/bin/bash
set -euo pipefail

mkdir -p /etc/hauler

mv haul.tar.zst /etc/hauler/haul.tar.zst
mv hauler /usr/bin/hauler

cat <<- EOF > /etc/systemd/system/registry.service
  [Unit]
  Description=Hauler Load and Serve
  After=network.target

  [Service]
  Type=simple
  User=root
  WorkingDirectory=/etc/hauler
  ExecStartPre=/usr/bin/hauler store load haul.tar.zst
  ExecStart=/usr/bin/hauler store serve
  Restart=on-failure

  [Install]
  WantedBy=multi-user.target
EOF

systemctl enable registry.service