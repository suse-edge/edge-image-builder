#!/bin/bash
set -euo pipefail

#  Template Fields
#  WorkDir     - directory from where this script will be running
#  ImgPath     - path to the image that will be prepared
#  ImgType     - type of the image (either .iso, or .raw)
#  ArchiveName - name of the virtual disk archive that will be created from this image

WORK_DIR={{.WorkDir}}
IMG_PATH={{.ImgPath}}

{{ if eq .ImgType "iso" -}}
xorriso -osirrox on -indev $IMG_PATH extract / $WORK_DIR/iso-root/

ISO_ROOT=$WORK_DIR/iso-root/
cd $ISO_ROOT

ISO_SQUASHFS=`find $ISO_ROOT -name "*.squashfs"`
unsquashfs $ISO_SQUASHFS 

UNSQUASHFS_DIR=$WORK_DIR/iso-root/squashfs-root
cd $UNSQUASHFS_DIR

RAW_FILE=`find $UNSQUASHFS_DIR -name "*.raw"`
virt-tar-out -a $RAW_FILE / - | gzip --best > $WORK_DIR/{{.ArchiveName}}
{{ else }}
virt-tar-out -a $IMG_PATH / - | gzip --best > $WORK_DIR/{{.ArchiveName}}
{{ end }}