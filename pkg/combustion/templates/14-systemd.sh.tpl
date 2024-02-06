#!/bin/bash
set -euo pipefail

{{ range .Disable }}
  systemctl disable {{ . }}
  systemctl mask {{ . }}
{{ end }}

{{ range .Enable }}
  systemctl enable {{ . }}
{{ end }}