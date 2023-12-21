#!/bin/bash
set -euo pipefail

mount /var
mkdir -p /var/lib/rancher/rke2/agent/images/
cp {{ .ImagesPath }}/* /var/lib/rancher/rke2/agent/images/
umount /var

{{- if .ConfigFile }}
mkdir -p /etc/rancher/rke2/
cp {{ .ConfigFile }} /etc/rancher/rke2/config.yaml
{{- end }}

{{- if .NodeType }}
export INSTALL_RKE2_TYPE={{ .NodeType }}
{{- end }}

export INSTALL_RKE2_TAR_PREFIX=/opt/rke2
export INSTALL_RKE2_ARTIFACT_PATH={{ .InstallPath }}

./rke2_installer.sh

echo "export KUBECONFIG=/etc/rancher/rke2/rke2.yaml" >> ~/.bashrc
echo "export PATH=${PATH}:/var/lib/rancher/rke2/bin" >> ~/.bashrc

systemctl enable rke2-{{ or .NodeType "server" }}.service
