#!/bin/bash
set -euo pipefail

cat <<EOF >/etc/systemd/system/firstboot-timesync.service
[Unit]
Description=Attempt NTP timesync to occur before starting Kubernetes services
Requires=chronyd.service
Wants=network-online.target
After=network-online.target
After=chrony-wait.service
Before=rke2-server.service
Before=rke2-agent.service
Before=k3s.service

[Service]
User=root
Type=oneshot
ExecStart=/usr/bin/echo "[INFO] Either reached 180s timeout or was successful in timesync before starting system services."
RemainAfterExit=true

[Install]
WantedBy=multi-user.target
EOF

systemctl enable chrony-wait
systemctl enable firstboot-timesync.service

echo "[WARN]: Waiting up to 180s to synchronise system clock with available NTP sources."
