#!/bin/bash
set -euo pipefail

./{{ .NMCExecutablePath }} apply --config-dir {{ .ConfigDir }}
