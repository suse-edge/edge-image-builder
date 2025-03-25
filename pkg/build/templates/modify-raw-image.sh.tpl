#!/bin/bash
set -euo pipefail

#  Template Fields
#  ImagePath                 - Full path to the image to modify
#  CombustionDir             - Full path to the combustion directory
#  ArtefactsDir              - Full path to the artefacts directory
#  ConfigureGRUB             - Contains the guestfish command lines to run to manipulate GRUB configuration.
#                              If there is no specific GRUB configuration to do, this will be an empty string.
#  ConfigureCombustion       - If true, the combustion and artefacts directories will be included in the raw image
#  RenameFilesystem          - If true, the filesystem of the image will be renamed (see below for information
#                            on why this is needed)
#  Arch                      - The architecture of the image to be built
#  LUKSKey                   - The key necessary for modifying encrypted raw images
#  ExpandEncryptedPartition  - Optionally enables expanding the encrypted partition before boot
#
# Guestfish Command Documentation: https://libguestfs.org/guestfish.1.html

# In x86_64, the default root partition is the third partition
ROOT_PART=/dev/sda3

# Make the necessary adaptations for aarch64
if [[ {{ .Arch }} == "aarch64" ]]; then
	if ! test -f /dev/kvm; then
		export LIBGUESTFS_BACKEND_SETTINGS=force_tcg
	fi
	ROOT_PART=/dev/sda2
fi

# Set the LUKS key flag for encrypted images
LUKSFLAG=""
{{ if .LUKSKey }}
LUKSFLAG="--key all:key:{{ .LUKSKey }}"
{{ end }}

# Test the block size of the base image and adapt to suit either 512/4096 byte images
BLOCKSIZE=512
if ! guestfish -i --blocksize=$BLOCKSIZE -a {{.ImagePath}} $LUKSFLAG echo "[INFO] 512 byte sector check successful."; then
        echo "[WARN] Failed to access image with 512 byte sector size, trying 4096 bytes."
        BLOCKSIZE=4096
fi

# Resize the raw disk image to accommodate the users desired raw disk image size
# This is also required if embedding content into /combustion, especially for airgap.
# Should *only* execute if the user is building a raw disk image.
{{ if ne .DiskSize "" -}}
truncate -r {{.ImagePath}} {{.ImagePath}}.expanded
truncate -s {{.DiskSize}} {{.ImagePath}}.expanded
virt-resize --expand $ROOT_PART {{.ImagePath}} {{.ImagePath}}.expanded
cp {{.ImagePath}}.expanded {{.ImagePath}}
rm -f {{.ImagePath}}.expanded
{{ end }}

guestfish --blocksize=$BLOCKSIZE --format=raw --rw -a {{.ImagePath}} $LUKSFLAG -i <<'EOF'
  # Enables write access to the read only filesystem
  sh "btrfs property set / ro false"

  {{ if .ExpandEncryptedPartition }}
  sh "btrfs filesystem resize max /"
  {{ end }}

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
