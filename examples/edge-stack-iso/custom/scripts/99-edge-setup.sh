#!/bin/bash

# Copy the scripts from combustion to the final location
mkdir -p /opt/edge/bin/
for script in basic-setup.sh rancher.sh metal3.sh; do
	cp ${script} /opt/edge/bin/
done

# Copy the systemd unit file and enable it at boot
cp edge-stack-setup.service /etc/systemd/system/edge-stack-setup.service
systemctl enable edge-stack-setup.service