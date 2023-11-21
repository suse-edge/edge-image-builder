#!/bin/bash
set -euo pipefail

# Redirect output to the console
exec > >(exec tee -a /dev/tty0) 2>&1

cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1
