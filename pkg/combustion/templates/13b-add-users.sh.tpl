#!/bin/bash
set -euo pipefail

# Without this, the script will run successfully during combustion, but when /home
# is mounted it will hide the /home used during these user creations.
mount /home

{{- range $user := . }}

{{- /* Non-root users */}}
{{- if (ne $user.Username "root") }}
PRIMARY_GROUP=""
SECONDARY_GROUPS=""
{{- if $user.PrimaryGroup }}
PRIMARY_GROUP="-g {{ $user.PrimaryGroup }}"
{{- end }}
{{- if $user.SecondaryGroups }}
SECONDARY_GROUPS="-G {{ join $user.SecondaryGroups "," }}"
{{- end }}
useradd $PRIMARY_GROUP $SECONDARY_GROUPS {{$user.Username}}

{{- if $user.EncryptedPassword }}
echo '{{$user.Username}}:{{$user.EncryptedPassword}}' | chpasswd -e
{{- end }}

{{- range $user.SSHKeys }}
mkdir -pm700 /home/{{$user.Username}}/.ssh/
echo '{{.}}' >> /home/{{$user.Username}}/.ssh/authorized_keys
chown -R {{$user.Username}} /home/{{$user.Username}}/.ssh
{{- end }}

{{- else }}

{{- /* Root user */}}
{{- if $user.EncryptedPassword }}
echo '{{$user.Username}}:{{$user.EncryptedPassword}}' | chpasswd -e
{{- end }}

{{- range $user.SSHKeys }}
mkdir -pm700 /{{$user.Username}}/.ssh/
echo '{{.}}' >> /{{$user.Username}}/.ssh/authorized_keys
{{- end }}

{{- end }}

{{- end }}
