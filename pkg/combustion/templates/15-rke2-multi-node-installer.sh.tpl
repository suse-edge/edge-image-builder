#!/bin/bash
set -euo pipefail

declare -A hosts

{{- range .Nodes }}
hosts[{{ .Hostname }}]={{ .Type }}
{{- end }}

HOSTNAME=$(cat /etc/hostname)
NODETYPE=${hosts[$HOSTNAME]}

if [ ! "$NODETYPE" ]; then
    echo "Could not identify whether host '$HOSTNAME' is an RKE2 server or agent";
    exit 1;
fi

mount /var
mkdir -p /var/lib/rancher/rke2/agent/images/
cp {{ .ImagesPath }}/* /var/lib/rancher/rke2/agent/images/

CONFIGFILE=$NODETYPE.yaml

if [ "$HOSTNAME" = {{ .Initialiser }} ]; then
    CONFIGFILE={{ .InitialiserConfigFile }}

    mkdir -p /var/lib/rancher/rke2/server/manifests/
    cp {{ .HAManifest }} /var/lib/rancher/rke2/server/manifests/{{ .HAManifest }}
fi

umount /var

mkdir -p /etc/rancher/rke2/
cp $CONFIGFILE /etc/rancher/rke2/config.yaml

export INSTALL_RKE2_TAR_PREFIX=/opt/rke2
export INSTALL_RKE2_ARTIFACT_PATH={{ .InstallPath }}

./rke2_installer.sh

systemctl enable rke2-$NODETYPE.service
