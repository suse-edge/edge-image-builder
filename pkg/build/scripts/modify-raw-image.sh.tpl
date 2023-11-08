#!/bin/bash
set -euo pipefail

#  Template Fields
#  OutputImage   - Full path to the image to modify
#  CombustionDir - Full path to the combustion directory

#  Guestfish Commands Explanation
#
#  sh "btrfs property set / ro false"
#  - Enables write access to the read only filesystem
#
#  copy-in __.CombustionDir__ /
#  - Copies the combustion directory into the root of the image
#
#  sh "btrfs filesystem label / INSTALL"
#  - As of Oct 25, 2023, combustion only checks volumes of certain names for the
#    /combustion directory. The SLE Micro raw image sets the root partition name to
#    "ROOT", which isn't one of the checked volume names. This line changes the
#    label to "INSTALL" (the same as the ISO installer uses) so it's picked up
#    when combustion runs.
#
#  sh "btrfs property set / ro true"
#  - Resets the filesystem to read only

guestfish --rw -a {{.OutputImage}} -i <<'EOF'
  download /boot/grub2/grub.cfg /tmp/grub.cfg
  ! sed -i 's/ignition.platform.id=metal/ignition.platform.id={{.Platform}}/' /tmp/grub.cfg
  sh "btrfs property set / ro false"
  upload /tmp/grub.cfg /boot/grub2/grub.cfg
  copy-in {{.CombustionDir}} /
  sh "btrfs filesystem label / INSTALL"
  sh "btrfs property set / ro true"
EOF
