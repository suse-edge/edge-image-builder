#!/bin/bash
set -euo pipefail

#  Template Fields
#  RepoName - name of the air-gapped repository
#  RepoPath - path to the airgapped repository relative to the combustion dir in the image
#  PKGList  - list of packages that will be installed

{{ if ne .RepoName "" -}}
zypper ar file://{{.RepoPath}} {{.RepoName}}
zypper --no-gpg-checks install -r {{.RepoName}} -y --force-resolution --auto-agree-with-licenses {{.PKGList}}
zypper rr {{.RepoName}}
{{ else }}
zypper --no-gpg-checks install -y --force-resolution --auto-agree-with-licenses {{.PKGList}}
{{ end }}