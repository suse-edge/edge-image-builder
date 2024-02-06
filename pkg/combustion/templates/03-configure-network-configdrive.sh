#!/bin/bash
set -euo pipefail

# Attempt to statically configure a nic in the case where we find a network_data.json
# In a configuration drive

CONFIG_DRIVE=$(blkid --label config-2)
if [ -z "${CONFIG_DRIVE}" ]; then
  echo "No config-2 device found, skipping network configuration"
  exit 0
fi

mount -o ro $CONFIG_DRIVE /mnt

NETWORK_DATA_FILE="/mnt/openstack/latest/network_data.json"

if [ ! -f "${NETWORK_DATA_FILE}" ]; then
  echo "No ${NETWORK_DATA_FILE} found, skipping network configuration"
  umount /mnt
  exit 0
fi

mkdir -p /tmp/nmc/{desired,generated}
cp ${NETWORK_DATA_FILE} /tmp/nmc/desired/hostname.yaml
umount /mnt

nmc generate --config-dir /tmp/nmc/desired --output-dir /tmp/nmc/generated
nmc apply --config-dir /tmp/nmc/generated

systemctl restart NetworkManager
