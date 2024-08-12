#!/bin/bash
set -euo pipefail

mount /var

mkdir -p /var/lib/rancher/k3s/agent/images/
cp {{ .imagesPath }}/* /var/lib/rancher/k3s/agent/images/

umount /var

{{- if .manifestsPath }}
mkdir -p /opt/eib-k8s/manifests
cp {{ .manifestsPath }}/* /opt/eib-k8s/manifests/

cat <<- 'EOF' > /opt/eib-k8s/create_manifests.sh
#!/bin/bash
failed=false

for file in /opt/eib-k8s/manifests/*; do
    output=$(/opt/bin/kubectl create -f "$file" --kubeconfig=/etc/rancher/k3s/k3s.yaml 2>&1)

    if [ $? != 0 ]; then
        if [[ "$output" != *"AlreadyExists"* ]]; then
            failed=true
        fi
    fi
    echo "$output"
done

if [ $failed = "true" ]; then
    exit 1
fi
EOF

chmod +x /opt/eib-k8s/create_manifests.sh

cat <<- EOF > /etc/systemd/system/kubernetes-resources-install.service
[Unit]
Description=Kubernetes Resources Install
Requires=k3s.service
PartOf=k3s.service
After=k3s.service
ConditionPathExists=/opt/bin/kubectl
ConditionPathExists=/etc/rancher/k3s/k3s.yaml

[Install]
WantedBy=multi-user.target

[Service]
Type=oneshot
Restart=on-failure
RestartSec=60
ExecStartPre=/bin/sh -c 'until systemctl is-active --quiet k3s.service; do sleep 10; done'
ExecStart=/opt/eib-k8s/create_manifests.sh
# Disable the service and clean up
ExecStartPost=/bin/sh -c "systemctl disable kubernetes-resources-install.service"
ExecStartPost=rm -f /etc/systemd/system/kubernetes-resources-install.service
ExecStartPost=rm -rf /opt/eib-k8s
EOF

systemctl enable kubernetes-resources-install.service
{{- end }}

{{- if and .apiVIP .apiHost }}
echo "{{ .apiVIP }} {{ .apiHost }}" >> /etc/hosts
{{- end }}

mkdir -p /etc/rancher/k3s/
cp {{ .configFilePath }}/{{ .configFile }} /etc/rancher/k3s/config.yaml

if [ -f {{ .registryMirrors }} ]; then
cp {{ .registryMirrors }} /etc/rancher/k3s/registries.yaml
fi

export INSTALL_K3S_SKIP_DOWNLOAD=true
export INSTALL_K3S_SKIP_START=true
export INSTALL_K3S_BIN_DIR=/opt/bin

mkdir -p $INSTALL_K3S_BIN_DIR
cp {{ .binaryPath }} $INSTALL_K3S_BIN_DIR/k3s
chmod +x $INSTALL_K3S_BIN_DIR/k3s

sh {{ .installScript }}
