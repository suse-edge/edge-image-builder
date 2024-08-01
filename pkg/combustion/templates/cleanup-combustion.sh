#!/bin/bash
set -euo pipefail

rm -r /combustion

if test -d /artefacts; then
  rm -r /artefacts
fi