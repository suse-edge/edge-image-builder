#!/bin/bash
set -euo pipefail

mount /var
mkdir -p /var/lib/rancher/rke2/agent/images/
cp {{ .ImagesPath }}/* /var/lib/rancher/rke2/agent/images/

{{- if .VIPManifest }}
mkdir -p /var/lib/rancher/rke2/server/manifests/
cp {{ .VIPManifest }} /var/lib/rancher/rke2/server/manifests/{{ .VIPManifest }}
{{- end }}
umount /var

{{- if and .Network.APIVIP .Network.APIHost }}
echo "{{ .Network.APIVIP }} {{ .Network.APIHost }}" >> /etc/hosts
{{- end }}

mkdir -p /etc/rancher/rke2/
cp {{ .ConfigFile }} /etc/rancher/rke2/config.yaml

export INSTALL_RKE2_TAR_PREFIX=/opt/rke2
export INSTALL_RKE2_ARTIFACT_PATH={{ .InstallPath }}

./rke2_installer.sh

systemctl enable rke2-server.service
