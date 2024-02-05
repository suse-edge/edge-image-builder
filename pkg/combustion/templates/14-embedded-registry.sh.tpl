#!/bin/bash
set -euo pipefail

mkdir /opt/hauler

mount /usr/local
mv {{ .RegistryDir }}/{{ .EmbeddedRegistryTar }} /opt/hauler/
mv hauler /usr/local/bin/hauler
umount /usr/local


# This serves the hauler registry artifacts as OCI artifacts
cat <<- EOF > /etc/systemd/system/eib-embedded-registry.service
  [Unit]
  Description=Load and Serve Embedded Registry
  After=network.target

  [Service]
  Type=simple
  User=root
  WorkingDirectory=/opt/hauler
  ExecStartPre=/usr/local/bin/hauler store load {{ .EmbeddedRegistryTar }}
  ExecStart=/usr/local/bin/hauler store serve registry -p {{ .RegistryPort }}
  Restart=on-failure

  [Install]
  WantedBy=multi-user.target
EOF
systemctl enable eib-embedded-registry.service

# This serves the hauler registry artifacts as a file server
{{- if .ChartsDir }}
cat <<- EOF > /etc/systemd/system/eib-embedded-fileserver.service
  [Unit]
  Description=Load and Serve Embedded File Server
  After=network.target

  [Service]
  Type=simple
  User=root
  WorkingDirectory=/opt/hauler
  ExecStartPre=/bin/sleep 30
  ExecStart=/usr/local/bin/hauler store serve fileserver -p {{ .FileServerPort }}
  Restart=on-failure

  [Install]
  WantedBy=multi-user.target
EOF
systemctl enable eib-embedded-fileserver.service
{{- end }}


# This adds a service manifest to /var/lib/rancher/*/server/manifests
# This allows the Helm resources to access the hauler registry running on the host
{{- if .ChartsDir }}
HOST_IP=$(ip a show eth0 | awk '/inet / {print $2}' | cut -d/ -f1)

mount /var
mkdir -p /var/lib/rancher/{{ .K8sType }}/server/manifests/

cat <<- EOF > /var/lib/rancher/{{ .K8sType }}/server/manifests/registry-svc.yaml
apiVersion: v1
kind: Service
metadata:
  name: hauler-registry
  namespace: default
spec:
  ports:
  - protocol: TCP
    port: 80
    targetPort: {{ .FileServerPort }}
---
apiVersion: v1
kind: Endpoints
metadata:
  name: hauler-registry
  namespace: default
subsets:
  - addresses:
      - ip: $HOST_IP
    ports:
      - port: {{ .FileServerPort }}
EOF

umount /var
{{- end }}