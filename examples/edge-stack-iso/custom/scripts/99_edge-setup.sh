#!/bin/bash

# Copy the basic setup script from combustion to the final location
mkdir -p /opt/edge/bin/
cp basic-setup.sh /opt/edge/bin/
chmod a+x /opt/edge/bin/basic-setup.sh

# Same for rancher
cp rancher.sh /opt/edge/bin/
chmod a+x /opt/edge/bin/rancher.sh
# Same for metal3
cp metal3.sh /opt/edge/bin/
chmod a+x /opt/edge/bin/metal3.sh

# Copy the systemd unit file
cp edge-stack-setup.service /etc/systemd/system/edge-stack-setup.service
systemctl enable edge-stack-setup.service