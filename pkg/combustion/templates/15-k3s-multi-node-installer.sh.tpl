#!/bin/bash
set -euo pipefail

declare -A hosts

{{- range .nodes }}
hosts[{{ .Hostname }}]={{ .Type }}
{{- end }}

HOSTNAME=$(cat /etc/hostname)
if [ ! "$HOSTNAME" ]; then
    echo "ERROR: Could not identify whether the host is a k3s server or agent due to missing hostname"
    exit 1
fi

NODETYPE="${hosts[$HOSTNAME]:-none}"
if [ "$NODETYPE" = "none" ]; then
    echo "ERROR: Could not identify whether host '$HOSTNAME' is a k3s server or agent"
    exit 1
fi

mount /var

mkdir -p /var/lib/rancher/k3s/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/k3s/agent/images/

CONFIGFILE=$NODETYPE.yaml

if [ "$HOSTNAME" = {{ .initialiser }} ]; then
    CONFIGFILE={{ .initialiserConfigFile }}

    mkdir -p /var/lib/rancher/k3s/server/manifests/
    cp {{ .vipManifest }} /var/lib/rancher/k3s/server/manifests/{{ .vipManifest }}

    {{- if .manifestsPath }}
    cp {{ .manifestsPath }}/* /var/lib/rancher/k3s/server/manifests/
    {{- end }}
fi

umount /var

{{- if and .apiVIP .apiHost }}
echo "{{ .apiVIP }} {{ .apiHost }}" >> /etc/hosts
{{- end }}

mkdir -p /etc/rancher/k3s/
cp $CONFIGFILE /etc/rancher/k3s/config.yaml

{{- if .manifestsPath }}
cp {{ .registryMirrors }} /etc/rancher/k3s/registries.yaml
{{- end }}

export INSTALL_K3S_EXEC=$NODETYPE
export INSTALL_K3S_SKIP_DOWNLOAD=true
export INSTALL_K3S_SKIP_START=true

mount /usr/local

chmod +x {{ .binaryPath }}
cp {{ .binaryPath }} /usr/local/bin/k3s

./k3s_installer.sh

umount /usr/local
