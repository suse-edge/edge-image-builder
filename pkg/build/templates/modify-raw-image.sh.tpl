#!/bin/bash
set -euo pipefail

#  Template Fields
#  ImagePath           - Full path to the image to modify
#  CombustionDir       - Full path to the combustion directory
#  ArtefactsDir        - Full path to the artefacts directory
#  ConfigureGRUB       - Contains the guestfish command lines to run to manipulate GRUB configuration.
#                        If there is no specific GRUB configuration to do, this will be an empty string.
#  ConfigureCombustion - If true, the combustion and artefacts directories will be included in the raw image
#  RenameFilesystem    - If true, the filesystem of the image will be renamed (see below for information
#                        on why this is needed)
#
# Guestfish Command Documentation: https://libguestfs.org/guestfish.1.html

# Test the block size of the base image and adapt to suit either 512/4096 byte images
BLOCKSIZE=512
if ! guestfish -i --blocksize=$BLOCKSIZE -a {{.ImagePath}} echo "[INFO] 512 byte sector check successful."; then
        echo "[WARN] Failed to access image with 512 byte sector size, trying 4096 bytes."
        BLOCKSIZE=4096
fi

# Resize the raw disk image to accommodate the users desired raw disk image size
# This is also required if embedding content into /combustion, especially for airgap.
# Should *only* execute if the user is building a raw disk image.
{{ if ne .DiskSize "" -}}
truncate -r {{.ImagePath}} {{.ImagePath}}.expanded
truncate -s {{.DiskSize}} {{.ImagePath}}.expanded
virt-resize --expand /dev/sda3 {{.ImagePath}} {{.ImagePath}}.expanded
cp {{.ImagePath}}.expanded {{.ImagePath}}
rm -f {{.ImagePath}}.expanded
{{ end }}

guestfish --blocksize=$BLOCKSIZE --format=raw --rw -a {{.ImagePath}} -i <<'EOF'
  # Enables write access to the read only filesystem
  sh "btrfs property set / ro false"

  {{ if ne .ConfigureGRUB "" }}
  {{ .ConfigureGRUB }}
  {{ end }}

  {{ if .ConfigureCombustion }}
  copy-in {{.CombustionDir}} /
  copy-in {{.ArtefactsDir}} /
  {{ end }}

  {{ if .RenameFilesystem }}
  # As of Oct 25, 2023, combustion only checks volumes of certain names for the
  # /combustion directory. The SLE Micro raw image sets the root partition name to
  # "ROOT", which isn't one of the checked volume names. This line changes the
  # label to "INSTALL" (the same as the ISO installer uses) so it's picked up
  # when combustion runs.
  sh "btrfs filesystem label / INSTALL"
  {{ end }}

  # Resets the filesystem to read only
  sh "btrfs property set / ro true"
EOF
