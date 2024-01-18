#!/bin/bash
set -euo pipefail

./nmc apply --config-dir {{ .ConfigDir }}
