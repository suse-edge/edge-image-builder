#!/bin/bash
set -euo pipefail

#  Template Fields
#  RegCode      - scc.suse.com registration code
#  AddRepo      - additional third-party repositories that will be used in the resolution process
#  CacheDir     - zypper cache directory where all rpm dependencies will be downloaded to
#  PKGList      - list of packages for which to do the dependency resolution
#  LocalRPMList - list of local RPMs for which dependency resolution has to be done
#  LocalGPGList - list of local GPG keys that will be imported in the resolver image
#  NoGPGCheck   - when set to true skips the GPG validation for all third-party repositories and local RPMs

{{ if ne .RegCode "" }}
suseconnect -r {{ .RegCode }}
SLE_SP=$(cat /etc/rpm/macros.sle | awk '/sle/ {print $2};' | cut -c4) && suseconnect -p PackageHub/15.$SLE_SP/x86_64
zypper ref
trap "suseconnect -d" EXIT
{{ end -}}

{{- range $index, $repo := .AddRepo }}

{{- $gpgCheck := "" -}}
{{- if $.NoGPGCheck -}}
{{ $gpgCheck = "--no-gpgcheck" }}
{{- else if .Unsigned -}}
{{ $gpgCheck = "--gpgcheck-allow-unsigned-repo" }}
{{- end -}}

zypper ar {{ $gpgCheck }} -f {{ .URL }} addrepo {{- $index }}

{{ end -}}

{{ if and .LocalGPGList (not .NoGPGCheck) }}
rpm --import {{ .LocalGPGList }}
{{ end -}}

{{ if and .LocalRPMList (not .NoGPGCheck) }}
rpm -Kv {{ .LocalRPMList }}
{{ end -}}

zypper \
  --pkg-cache-dir {{.CacheDir}} \
  --gpg-auto-import-keys \
  {{ if .NoGPGCheck -}}
  --no-gpg-checks \
  {{ end -}}
  install -y \
  --download-only \
  --force-resolution \
  --auto-agree-with-licenses \
  --allow-vendor-change \
  -n {{.PKGList}} {{.LocalRPMList}}

touch {{.CacheDir}}/zypper-success