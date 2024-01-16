#!/bin/bash
set -euo pipefail

echo "KEYMAP={{ or .Keymap "us" }}" >> /etc/vconsole.conf
