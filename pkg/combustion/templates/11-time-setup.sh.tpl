#!/bin/bash
set -euo pipefail

{{ if .Timezone -}}
ln -sf /usr/share/zoneinfo/{{ .Timezone }} /etc/localtime
{{ end -}}

{{ if or (gt (len .Pools) 0) (gt (len .Servers) 0) }}
rm -f /etc/chrony.d/pool.conf
{{ end -}}

{{ range .Pools -}}
echo "pool {{ . }} iburst" >> /etc/chrony.d/eib-sources.conf
{{ end -}}

{{ range .Servers -}}
echo "server {{ . }} iburst" >> /etc/chrony.d/eib-sources.conf
{{ end -}}

{{ if .ForceWait -}}
# Create a simple systemd OneShot service that depends on networking and chrony-wait
# (a service that forces a synchronisation of local time with the available NTP sources
# but has a 180s timeout) and one that must complete before k3s/rke2 start. This temporary
# systemd unit enables us to wait on the chrony sync *without* modifying the chrony-wait
# service or the default k3s/rke2 systemd unit files. The systemd unit file needs to
# execute something, so we echo out to syslog that we've either reached the timeout, or
# the synchronisation was completed successfully, whichever comes first.
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

# Print to the console that we're pausing boot whilst the chrony-wait service executes.
# If this happens immediately then this will likely skip by, but if NTP is unavailable
# then it makes it clear to the user why the system is pausing.
echo "[WARN]: Waiting up to 180s to synchronise system clock with available NTP sources."
{{ end -}}
