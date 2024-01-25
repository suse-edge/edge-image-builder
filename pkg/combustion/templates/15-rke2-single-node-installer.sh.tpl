#!/bin/bash
set -euo pipefail

mount /var
mkdir -p /var/lib/rancher/rke2/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/rke2/agent/images/

{{- if .vipManifest }}
mkdir -p /var/lib/rancher/rke2/server/manifests/
cp {{ .vipManifest }} /var/lib/rancher/rke2/server/manifests/{{ .vipManifest }}
{{- end }}
umount /var

{{- if and .apiVIP .apiHost }}
echo "{{ .apiVIP }} {{ .apiHost }}" >> /etc/hosts
{{- end }}

mkdir -p /etc/rancher/rke2/
cp {{ .configFile }} /etc/rancher/rke2/config.yaml

export INSTALL_RKE2_TAR_PREFIX=/opt/rke2
export INSTALL_RKE2_ARTIFACT_PATH={{ .installPath }}

./rke2_installer.sh

systemctl enable rke2-server.service
