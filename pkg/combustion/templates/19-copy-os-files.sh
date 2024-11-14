#!/bin/bash
set -euo pipefail

mount /var
cp -R ./os-files/* /
umount /var