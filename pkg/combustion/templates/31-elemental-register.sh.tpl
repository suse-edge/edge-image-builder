#!/bin/bash
set -euo pipefail

mkdir -p /etc/elemental
cp ./{{ .ConfigFile }} /etc/elemental/config.yaml

# Enable systemd based Elemental registration
# Register --no-toolkit disables OS management in Rancher
cat <<- EOF > /etc/systemd/system/elemental-register-systemd.service
[Unit]
Description=Elemental Register Install via Systemd
Wants=network-online.target
After=network-online.target
ConditionPathExists=!/etc/rancher/elemental/agent/elemental_connection.json

[Install]
WantedBy=network-online.target

[Service]
EnvironmentFile=-/etc/sysconfig/proxy
Type=oneshot
ExecStart=/usr/sbin/elemental-register --debug --config-path /etc/elemental/config.yaml --state-path /etc/elemental/state.yaml --install --no-toolkit
ExecStartPost=/usr/bin/cp /var/lib/elemental/agent/elemental_connection.json /etc/rancher/elemental/agent
Restart=on-failure
RestartSec=10
EOF

# Enable elemental-system-agent
# On SLE Micro /var/lib is not persistent, so we copy elemental_connection.json in ExecStartPre
cat <<- EOF > /etc/systemd/system/elemental-system-agent.service
[Unit]
Description=Elemental System Agent
Documentation=https://github.com/rancher/system-agent
Wants=network-online.target
After=network-online.target
After=time-sync.target

[Install]
WantedBy=multi-user.target

[Service]
Type=simple
Restart=always
RestartSec=5s
StandardOutput=journal
StandardError=journal
Environment="CATTLE_AGENT_CONFIG=/etc/rancher/elemental/agent/config.yaml"
ExecStartPre=/bin/sh -c "mkdir -p /var/lib/elemental/agent && cp /etc/rancher/elemental/agent/elemental_connection.json /var/lib/elemental/agent"
ExecStart=/usr/sbin/elemental-system-agent sentinel
EOF

systemctl enable elemental-register-systemd.service || true
systemctl enable elemental-system-agent.service || true
