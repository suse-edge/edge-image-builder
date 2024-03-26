#!/bin/bash
set -euo pipefail

{{/* Template Fields */ -}}
{{/* RepoPath - path to the air-gapped repository that was created by the RPM resolver */ -}}
{{/* RepoName - name of the air-gapped repository that was created by the RPM resolver */ -}}
{{/* PKGList  - list of packages that will be installed */ -}}

zypper ar file://{{.RepoPath}}/{{.RepoName}} {{.RepoName}}
zypper --no-gpg-checks install -r {{.RepoName}} -y --force-resolution --auto-agree-with-licenses {{.PKGList}}
zypper rr {{.RepoName}}
