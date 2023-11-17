#!/bin/bash
set -euo pipefail

# Without this, the script will run successfully during combustion, but when /home
# is mounted it will hide the /home used during these user creations.
mount /home

{{ range . }}

{{/* Non-root users */}}
{{ if (ne .Username "root") }}
  useradd -m {{.Username}}
  {{ if .Password }}
    echo '{{.Username}}:{{.Password}}' | chpasswd -e
  {{ end }}
  {{ if .SSHKey }}
    mkdir -pm700 /home/{{.Username}}/.ssh/
    echo '{{.SSHKey}}' >> /home/{{.Username}}/.ssh/authorized_keys
    chown -R {{.Username}} /home/{{.Username}}/.ssh
  {{ end }}
{{ end }}

{{/* Root */}}
{{ if (eq .Username "root") }}
  {{ if .Password }}
    echo '{{.Username}}:{{.Password}}' | chpasswd -e
  {{ end }}
  {{ if .SSHKey }}
    mkdir -pm700 /{{.Username}}/.ssh/
    echo '{{.SSHKey}}' >> /{{.Username}}/.ssh/authorized_keys
  {{ end }}
{{ end }}

{{ end }}