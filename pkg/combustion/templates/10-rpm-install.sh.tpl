#!/bin/bash
set -euo pipefail

#  Template Fields
#  RepoName - name of the air-gapped repository that was created by the RPM resovler
#  PKGList  - list of packages that will be installed

zypper ar file:///dev/shm/combustion/config/{{.RepoName}} {{.RepoName}}
zypper --no-gpg-checks install -r {{.RepoName}} -y --force-resolution --auto-agree-with-licenses {{.PKGList}}
zypper rr {{.RepoName}}