#!/bin/bash
set -euo pipefail

mount /var
mkdir -p /var/lib/rancher/k3s/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/k3s/agent/images/
umount /var

export INSTALL_K3S_SKIP_DOWNLOAD=true
export INSTALL_K3S_SKIP_START=true
export INSTALL_K3S_BIN_DIR=/opt/k3s

mkdir -p $INSTALL_K3S_BIN_DIR
chmod +x {{ .binaryPath }}
cp {{ .binaryPath }} $INSTALL_K3S_BIN_DIR/k3s

./k3s_installer.sh
