#!/bin/bash
set -euo pipefail

echo "KEYMAP={{ .Keymap }}" >> /etc/vconsole.conf
