#!/bin/bash
set -euo pipefail

#  Template Fields
#  ImagePath           - Full path to the image to modify
#  CombustionDir       - Full path to the combustion directory
#  ConfigureGRUB       - Contains the guestfish command lines to run to manipulate GRUB configuration.
#                        If there is no specific GRUB configuration to do, this will be an empty string.
#  ConfigureCombustion - If true, the combustion directory will be included in the raw image
#  RenameFilesystem    - If true, the filesystem of the image will be renamed (see below for information
#                        on why this is needed)
#
# Guestfish Command Documentation: https://libguestfs.org/guestfish.1.html

guestfish --format=raw --rw -a {{.ImagePath}} -i <<'EOF'
  # Enables write access to the read only filesystem
  sh "btrfs property set / ro false"

  {{ if ne .ConfigureGRUB "" }}
  {{ .ConfigureGRUB }}
  {{ end }}

  {{ if .ConfigureCombustion }}
  copy-in {{.CombustionDir}} /
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
