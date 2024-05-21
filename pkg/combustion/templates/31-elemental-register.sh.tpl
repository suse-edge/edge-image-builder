#!/bin/bash
set -euo pipefail

mkdir -p /etc/elemental
cp ./{{ .ConfigFile }} /etc/elemental/config.yaml

# Enable systemd based Elemental registration
# Register --no-toolkit disables OS management in Rancher
cat <<- EOF > /etc/systemd/system/elemental-register-systemd.service
[Unit]
Description=Elemental Register Install via Systemd
Wants=network-online.target
After=network-online.target
ConditionPathExists=!/etc/rancher/elemental/agent/elemental_connection.json

[Install]
WantedBy=network-online.target

[Service]
EnvironmentFile=-/etc/sysconfig/proxy
Type=oneshot
ExecStart=/usr/sbin/elemental-register --debug --config-path /etc/elemental/config.yaml --state-path /etc/elemental/state.yaml --install --no-toolkit
ExecStartPost=/usr/bin/cp /var/lib/elemental/agent/elemental_connection.json /etc/rancher/elemental/agent
Restart=on-failure
RestartSec=10
EOF

# Enable elemental-system-agent
cat <<- EOF > /etc/systemd/system/elemental-system-agent.service
[Unit]
Description=Elemental System Agent
Documentation=https://github.com/rancher/system-agent
Wants=network-online.target
After=network-online.target
After=time-sync.target

[Install]
WantedBy=multi-user.target

[Service]
Type=simple
Restart=always
RestartSec=5s
StandardOutput=journal
StandardError=journal
Environment="CATTLE_AGENT_CONFIG=/etc/rancher/elemental/agent/config.yaml"
ExecStart=/usr/sbin/elemental-system-agent sentinel
EOF

cat <<- EOF > /etc/systemd/system/elemental-reset.path
[Path]
PathModified=/var/lib/elemental/.unmanaged_reset

[Install]
WantedBy=multi-user.target
EOF

cat <<- EOF > /etc/systemd/system/elemental-reset.service
[Unit]
Description=Elemental Reset for Unmanaged Hosts
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
ExecStart=/opt/edge/elemental_node_cleanup.sh -u
ExecStartPost=/usr/bin/rm -f /var/lib/elemental/.unmanaged_reset
EOF

systemctl enable elemental-reset.path || true
systemctl enable elemental-register-systemd.service || true
systemctl enable elemental-system-agent.service || true

mkdir -p /opt/edge/
cat <<- \EOF > /opt/edge/elemental_node_cleanup.sh
#!/usr/bin/env bash
# SUSE Edge Elemental Node Reset Script
# Copyright 2024 SUSE Software Solutions

# This script attempts to cleanup a node that has been deployed via Edge Image
# Builder with the integrations for Elemental registration; in other words,
# vanilla SLE Micro 5.5, *not* SLE Micro for Rancher (also known as Elemental
# Teal), that has used the "--no-toolkit" registration option.
#
# The default behaviour in Rancher/Elemental is that in the event that a
# cluster is deleted in Rancher, the Kubernetes cluster running on a node (or
# set of nodes) will not be automatically cleaned up; the cluster will be
# orphaned and will remain running. Furthermore, the Elemental MachineInventory
# will be removed, so it's no longer visible in the list of registered nodes.
#
# This script cleans up the installed Kubernetes cluster so no traces remain
# and forces a re-registration with the original Elemental registration config.
#
# WARNING: This script *will* cause data loss as it removes all Kubernetes
#          persistent data. There is also an unattended switch for automated
#          reset. You have been warned!

UNATTENDED=false

while getopts 'u' OPTION; do
    case "${OPTION}" in
        u)
            UNATTENDED=true
            ;;
    esac
done

if [ $UNATTENDED = "false" ] ;
then
    echo "============================================"
    echo "SUSE Edge Node Cleanup for Elemental Systems"
    echo -e "============================================\n"
    echo -n "WARNING: This script will remove all Kubernetes files and will"
    echo -e " cause data loss!\n"
    while true; do
            read -p "Are you sure you wish to proceed [y/N]? " yn
            case $yn in
                [Yy] ) break;;
                [Nn] ) exit;;
                * ) exit 0;;
            esac
        done
fi

# If we reach this point, we're deleting data and re-registering.

# Stop both the elemental and rancher-system-agents via systemd
systemctl kill --signal=SIGKILL elemental-system-agent
systemctl kill --signal=SIGKILL rancher-system-agent

# Kill and uninstall all rke2 services
if [ -x /opt/rke2/bin/rke2-uninstall.sh ];
then
    /opt/rke2/bin/rke2-killall.sh
    /opt/rke2/bin/rke2-uninstall.sh
fi

# Kill and uninstall all k3s services
if command -v k3s-killall.sh &> /dev/null; then k3s-killall.sh; fi
if command -v k3s-uninstall.sh &> /dev/null; then k3s-uninstall.sh; fi

# Remove the rancher-system-agent as this gets reinstalled via Elemental
if [ -x /opt/rancher-system-agent/bin/rancher-system-agent-uninstall.sh ];
then
    sh /opt/rancher-system-agent/bin/rancher-system-agent-uninstall.sh
    rm -rf /opt/rancher-system-agent
fi

# Clean up all old configuration directories and Elemental state
rm -rf /etc/rancher
rm -rf /var/lib/rancher
rm -f /etc/elemental/state.yaml

# Re-register the node via Elemental using the original Elemental config
# by restarting the Elemental registration service via systemd
systemctl restart elemental-register-systemd.service
EOF

chmod a+x /opt/edge/elemental_node_cleanup.sh
