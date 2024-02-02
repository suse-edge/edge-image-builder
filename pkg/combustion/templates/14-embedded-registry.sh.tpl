#!/bin/bash
set -euo pipefail

mkdir /opt/hauler

mount /usr/local
mv {{ .RegistryDir }}/{{ .EmbeddedRegistryTar }} /opt/hauler/
mv hauler /usr/local/bin/hauler
umount /usr/local

{{- if .ChartsDir }}
mount /var
mkdir -p /var/lib/rancher/rke2/server/manifests/static/charts
cp {{.ChartsDir }}/*  /var/lib/rancher/rke2/server/manifests/static/charts/
umount /var
{{- end }}

cat <<- EOF > /etc/systemd/system/eib-embedded-registry.service
  [Unit]
  Description=Load and Serve Embedded Registry
  After=network.target

  [Service]
  Type=simple
  User=root
  WorkingDirectory=/opt/hauler
  ExecStartPre=/usr/local/bin/hauler store load {{ .EmbeddedRegistryTar }}
  ExecStart=/usr/local/bin/hauler store serve registry -p {{ .Port }}
  Restart=on-failure

  [Install]
  WantedBy=multi-user.target
EOF

systemctl enable eib-embedded-registry.service