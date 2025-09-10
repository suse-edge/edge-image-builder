#!/bin/bash
set -euo pipefail

mkdir -p /opt/hauler
cp {{ .RegistryDir }}/hauler /opt/hauler/hauler
cp {{ .RegistryDir }}/*-{{ .RegistryTarSuffix }} /opt/hauler/

cat <<- 'EOF' > /opt/hauler/start-registry.sh
#!/bin/bash
set -euo pipefail

# Load all registry tar files
for file in /opt/hauler/*-{{ .RegistryTarSuffix }}; do
    [ -f "$file" ] && /opt/hauler/hauler store load -f "$file" --tempdir /opt/hauler
done

# Start the registry server
exec /opt/hauler/hauler store serve registry -p {{ .RegistryPort }}
EOF

chmod +x /opt/hauler/start-registry.sh

cat <<- EOF > /etc/systemd/system/eib-embedded-registry.service
[Unit]
Description=Load and Serve Embedded Registry
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/hauler
ExecStart=/opt/hauler/start-registry.sh
TimeoutStartSec=300
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

systemctl enable eib-embedded-registry.service
