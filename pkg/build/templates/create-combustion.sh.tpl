#!/bin/bash
set -euo pipefail

#  Template Fields
#  OutputImageFilename - Full path and name of the ISO to create
#  CombustionDir - Full path to the combustion directory to include in the new ISO
#  CombustionTmpPath - Full path to the temp location to assemble the combustion ISO and tar
#  ArtefactsDir        - Full path to the artefacts directory

mkdir -p {{ .CombustionTmpPath }}
cp -r {{ .CombustionDir }} {{ .CombustionTmpPath }}
cp -r {{ .ArtefactsDir }} {{ .CombustionTmpPath }}

mkisofs -J -o {{.OutputImageFilename}} -V COMBUSTION {{ .CombustionTmpPath }}