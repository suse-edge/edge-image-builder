#!/bin/bash
set -euo pipefail

mount /var
mount /usr/local
cp -R ./os-files/* /
umount /var
umount /usr/local
