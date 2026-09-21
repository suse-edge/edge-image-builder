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
#  Arch         - sets the architecture of the rpm packages to pull
#  EnableExtras - registers the SL-Micro-Extras repo for use in resolution

{{ if ne .RegCode "" }}
suseconnect -r {{ .RegCode }}
{{ if $.EnableExtras -}}
VERSION=$(

 OS_RELEASE_FILE="${1:-/etc/os-release}"

# Make sure the file actually exists before we try to read it.
if [[ ! -f "$OS_RELEASE_FILE" ]]; then
    echo "Error: file not found: $OS_RELEASE_FILE"
    exit 1
fi

# Step 1: try to read the new field first (SL Micro 6.2+).
new_version=$(grep '^SUSE_SUPPORT_PRODUCT_VERSION=' "$OS_RELEASE_FILE" | cut -d '=' -f2 | tr -d '"')

# Step 2: if that field wasn't found, fall back to the old field.
old_version=$(grep '^VERSION_ID=' "$OS_RELEASE_FILE" | cut -d '=' -f2 | tr -d '"')

if [[ -n "$new_version" ]]; then
    version="$new_version"
else
    version="$old_version"
fi

# Step 3: make sure we actually got something.
if [[ -z "$version" ]]; then
    echo "Error: could not find a version in $OS_RELEASE_FILE"
    exit 1
fi

# Step 4: check that it looks like a real version number, e.g. "6.2".
if [[ ! "$version" =~ ^[0-9]+(\.[0-9]+)*$ ]]; then
    echo "Error: found version '$version' but it doesn't look valid"
    exit 1
fi

echo "Version is: $version"
)

suseconnect -p SL-Micro-Extras/$VERSION/{{ .Arch }}
{{ end -}}
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

zypper ar {{ $gpgCheck }} -f --priority {{ .Priority }} {{ .URL }} addrepo {{- $index }}

{{ end -}}

{{ if and .LocalGPGList (not .NoGPGCheck) }}
rpm --import {{ .LocalGPGList }}
{{ end -}}

{{ if and .LocalRPMList (not .NoGPGCheck) }}
rpm -Kv {{ .LocalRPMList }}
{{ end -}}

mkdir -p {{.CacheDir}}

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
