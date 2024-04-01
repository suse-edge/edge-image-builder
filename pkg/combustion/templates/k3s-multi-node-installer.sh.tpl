#!/bin/bash
set -euo pipefail

declare -A hosts

{{- range .nodes }}
hosts[{{ .Hostname }}]={{ .Type }}
{{- end }}

HOSTNAME=$(cat /etc/hostname)
if [ ! "$HOSTNAME" ]; then
    HOSTNAME=$(cat /proc/sys/kernel/hostname)
    if [ ! "$HOSTNAME" ] || [ "$HOSTNAME" = "localhost.localdomain" ]; then
        echo "ERROR: Could not identify whether the host is a k3s server or agent due to missing hostname"
        exit 1
    fi
fi

NODETYPE="${hosts[$HOSTNAME]:-none}"
if [ "$NODETYPE" = "none" ]; then
    echo "ERROR: Could not identify whether host '$HOSTNAME' is a k3s server or agent"
    exit 1
fi

mount /var

mkdir -p /var/lib/rancher/k3s/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/k3s/agent/images/

CONFIGFILE={{ .configFilePath }}/$NODETYPE.yaml

if [ "$HOSTNAME" = {{ .initialiser }} ]; then
    CONFIGFILE={{ .configFilePath }}/{{ .initialiserConfigFile }}

    {{- if .manifestsPath }}
    mkdir -p /var/lib/rancher/k3s/server/manifests/
    cp {{ .manifestsPath }}/* /var/lib/rancher/k3s/server/manifests/
    {{- end }}
fi

umount /var

{{- if and .apiVIP .apiHost }}
echo "{{ .apiVIP }} {{ .apiHost }}" >> /etc/hosts
{{- end }}

mkdir -p /etc/rancher/k3s/
cp $CONFIGFILE /etc/rancher/k3s/config.yaml

if [ -f {{ .registryMirrors }} ]; then
cp {{ .registryMirrors }} /etc/rancher/k3s/registries.yaml
fi

export INSTALL_K3S_EXEC=$NODETYPE
export INSTALL_K3S_SKIP_DOWNLOAD=true
export INSTALL_K3S_SKIP_START=true
export INSTALL_K3S_BIN_DIR=/opt/bin

mkdir -p $INSTALL_K3S_BIN_DIR
cp {{ .binaryPath }} $INSTALL_K3S_BIN_DIR/k3s
chmod +x $INSTALL_K3S_BIN_DIR/k3s

sh {{ .installScript }}
