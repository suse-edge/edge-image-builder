#!/bin/bash
set -euo pipefail

# Use "|| true" in order to allow for DHCP configurations in cases where nmc fails
./nmc apply --config-dir {{ .ConfigDir }} || true
