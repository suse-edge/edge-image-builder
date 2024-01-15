#!/bin/bash
set -euo pipefail

#  Template Fields
#  RepoName - name of the air-gapped repository
#  RepoPath - path to the air-gapped repository relative to the combustion dir in the image
#  PKGList  - list of packages that will be installed

zypper ar file://{{.RepoPath}} {{.RepoName}}
zypper --no-gpg-checks install -r {{.RepoName}} -y --force-resolution --auto-agree-with-licenses {{.PKGList}}
zypper rr {{.RepoName}}