#!/bin/bash
set -euo pipefail

mkdir -p /etc/elemental
cp ./{{ .ConfigFile }} /etc/elemental/config.yaml

# Register --no-toolkit disables OS management
elemental-register --config-path /etc/elemental/config.yaml --state-path /etc/elemental/state.yaml --install --no-toolkit

# Enable elemental-system-agent
# On SLEMicro /var/lib is not persistent, so we copy elemental_connection.json in ExecStartPre
cp /var/lib/elemental/agent/elemental_connection.json /etc/rancher/elemental/agent
cat <<- EOF > /etc/systemd/system/elemental-system-agent.service
[Unit]
Description=Elemental System Agent
Documentation=https://github.com/rancher/system-agent
Wants=network-online.target
After=network-online.target
After=time-sync.target

[Install]
WantedBy=multi-user.target
Alias=elemental-system-agent.service

[Service]
Type=simple
Restart=always
RestartSec=5s
StandardOutput=journal+console
StandardError=journal+console
Environment="CATTLE_AGENT_CONFIG=/etc/rancher/elemental/agent/config.yaml"
ExecStartPre=/bin/sh -c "mkdir -p /var/lib/elemental/agent && cp /etc/rancher/elemental/agent/elemental_connection.json /var/lib/elemental/agent"
ExecStart=/usr/sbin/elemental-system-agent sentinel
EOF
systemctl enable elemental-system-agent.service || true