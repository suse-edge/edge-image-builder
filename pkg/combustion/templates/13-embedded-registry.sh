#!/bin/bash
set -euo pipefail

mkdir /opt/hauler

mv haul.tar.zst /opt/hauler/haul.tar.zst
mv hauler /usr/bin/hauler

cat <<- EOF > /etc/systemd/system/embedded-registry.service
  [Unit]
  Description=Load and Serve Embedded Registry
  After=network.target

  [Service]
  Type=simple
  User=root
  WorkingDirectory=/opt/hauler
  ExecStartPre=/usr/bin/hauler store load haul.tar.zst
  ExecStart=/usr/bin/hauler store serve
  Restart=on-failure

  [Install]
  WantedBy=multi-user.target
EOF

systemctl enable embedded-registry.service