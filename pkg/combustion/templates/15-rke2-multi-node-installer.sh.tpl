#!/bin/bash
set -euo pipefail

declare -A hosts

{{- range .nodes }}
hosts[{{ .Hostname }}]={{ .Type }}
{{- end }}

HOSTNAME=$(cat /etc/hostname)
if [ ! "$HOSTNAME" ]; then
    echo "ERROR: Could not identify whether the host is an RKE2 server or agent due to missing hostname"
    exit 1
fi

NODETYPE="${hosts[$HOSTNAME]:-none}"
if [ "$NODETYPE" = "none" ]; then
    echo "ERROR: Could not identify whether host '$HOSTNAME' is an RKE2 server or agent"
    exit 1
fi

mount /var
mkdir -p /var/lib/rancher/rke2/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/rke2/agent/images/


CONFIGFILE=$NODETYPE.yaml

if [ "$HOSTNAME" = {{ .initialiser }} ]; then
    CONFIGFILE={{ .initialiserConfigFile }}

    mkdir -p /var/lib/rancher/rke2/server/manifests/
    cp {{ .vipManifest }} /var/lib/rancher/rke2/server/manifests/{{ .vipManifest }}

    {{ - if .manifestsPath }}
    mkdir -p /var/lib/rancher/rke2/server/manifests/
    cp {{ .manifestsPath }}/* /var/lib/rancher/rke2/server/manifests/
    {{ - end }}

fi

umount /var

{{- if .apiHost }}
echo "{{ .apiVIP }} {{ .apiHost }}" >> /etc/hosts
{{- end }}

mkdir -p /etc/rancher/rke2/
cp $CONFIGFILE /etc/rancher/rke2/config.yaml

export INSTALL_RKE2_TAR_PREFIX=/opt/rke2
export INSTALL_RKE2_ARTIFACT_PATH={{ .installPath }}

./rke2_installer.sh

systemctl enable rke2-$NODETYPE.service
