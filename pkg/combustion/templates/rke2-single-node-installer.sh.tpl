#!/bin/bash
set -euo pipefail

mount /var

mkdir -p /var/lib/rancher/rke2/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/rke2/agent/images/

{{- if .manifestsPath }}
mkdir -p /var/lib/rancher/rke2/server/manifests/
cp {{ .manifestsPath }}/* /var/lib/rancher/rke2/server/manifests/
{{- end }}

umount /var

{{- if and .apiVIP .apiHost }}
echo "{{ .apiVIP }} {{ .apiHost }}" >> /etc/hosts
{{- end }}

mkdir -p /etc/rancher/rke2/
cp {{ .configFilePath }}/{{ .configFile }} /etc/rancher/rke2/config.yaml

if [ -f {{ .registryMirrors }} ]; then
cp {{ .registryMirrors }} /etc/rancher/rke2/registries.yaml
fi

export INSTALL_RKE2_TAR_PREFIX=/opt/rke2
export INSTALL_RKE2_ARTIFACT_PATH={{ .installPath }}

# Create the CNI directory, usually created and labelled by the
# rke2-selinux package, but isn't executed during combustion.
mkdir -p /opt/cni

sh {{ .installScript }}

systemctl enable rke2-server.service
