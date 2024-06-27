#!/bin/bash
set -euo pipefail

mount /var

mkdir -p /var/lib/rancher/rke2/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/rke2/agent/images/

umount /var

{{- if .manifestsPath }}
mkdir -p /opt/k8s/manifests
cp {{ .manifestsPath }}/* /opt/k8s/manifests/

cat <<- EOF > /etc/systemd/system/kubernetes-resources-install.service
[Unit]
Description=Kubernetes Resources Install
Requires=rke2-server.service
After=rke2-server.service
ConditionPathExists=/var/lib/rancher/rke2/bin/kubectl
ConditionPathExists=/etc/rancher/rke2/rke2.yaml

[Install]
WantedBy=multi-user.target

[Service]
Type=oneshot
Restart=on-failure
RestartSec=60
# Copy kubectl in order to avoid SELinux permission issues
ExecStartPre=cp /var/lib/rancher/rke2/bin/kubectl /opt/k8s/kubectl
ExecStart=/opt/k8s/kubectl apply -f /opt/k8s/manifests --kubeconfig /etc/rancher/rke2/rke2.yaml
# Disable the service and clean up
ExecStartPost=/bin/sh -c "systemctl disable kubernetes-resources-install.service"
ExecStartPost=rm -f /etc/systemd/system/kubernetes-resources-install.service
ExecStartPost=rm -rf /opt/k8s
EOF

systemctl enable kubernetes-resources-install.service
{{- end }}

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
