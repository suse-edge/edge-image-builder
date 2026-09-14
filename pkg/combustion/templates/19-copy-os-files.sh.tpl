#!/bin/bash
set -euo pipefail

mount /var
mount /usr/local
cp -R {{ .FilesPath }}/* /
umount /var
umount /usr/local
