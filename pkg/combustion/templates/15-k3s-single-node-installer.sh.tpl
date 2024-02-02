#!/bin/bash
set -euo pipefail

mount /var

mkdir -p /var/lib/rancher/k3s/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/k3s/agent/images/

{{- if .vipManifest }}
mkdir -p /var/lib/rancher/k3s/server/manifests/
cp {{ .vipManifest }} /var/lib/rancher/k3s/server/manifests/{{ .vipManifest }}
{{- end }}

{{- if .manifestsPath }}
mkdir -p /var/lib/rancher/k3s/server/manifests/
cp {{ .manifestsPath }}/* /var/lib/rancher/k3s/server/manifests/
{{- end }}

umount /var

{{- if and .apiVIP .apiHost }}
echo "{{ .apiVIP }} {{ .apiHost }}" >> /etc/hosts
{{- end }}

mkdir -p /etc/rancher/k3s/
cp {{ .configFile }} /etc/rancher/k3s/config.yaml

{{- if .manifestsPath }}
cp {{ .registryMirrors }} /etc/rancher/k3s/registries.yaml
{{- end }}

export INSTALL_K3S_SKIP_DOWNLOAD=true
export INSTALL_K3S_SKIP_START=true

mount /usr/local

chmod +x {{ .binaryPath }}
cp {{ .binaryPath }} /usr/local/bin/k3s

./k3s_installer.sh

umount /usr/local
