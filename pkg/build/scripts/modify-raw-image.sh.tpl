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
#  copy-in __.OutputImage__ /
#  - Copies the combustion directory into the root of the image
#  - __.OutputImage__ should be populated with the full path to the built combustion directory
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
  sh "btrfs property set / ro false"
  copy-in {{.CombustionDir}} /
  sh "btrfs filesystem label / INSTALL"
  sh "btrfs property set / ro true"
EOF
