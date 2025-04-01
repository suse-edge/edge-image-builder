#!/bin/bash
set -euo pipefail

#  Template Fields
#  WorkDir     - directory from where this script will be running
#  ImgPath     - path to the image that will be prepared
#  ImgType     - type of the image (either .iso, or .raw)
#  ArchiveName - name of the virtual disk archive that will be created from this image
#  LUKSKey     - The key necessary for modifying encrypted raw images

WORK_DIR={{.WorkDir}}
IMG_PATH={{.ImgPath}}

# Set the LUKS key flag for encrypted images
LUKSFLAG=""
{{ if .LUKSKey }}
LUKSFLAG="--key all:key:{{ .LUKSKey }}"
{{ end }}

# Make the necessarry adaptations for aarch64
if [[ {{ .Arch }} == "aarch64" ]]; then
	export LIBGUESTFS_BACKEND_SETTINGS=force_tcg
fi

{{ if eq .ImgType "iso" -}}
xorriso -osirrox on -indev $IMG_PATH extract / $WORK_DIR/iso-root/

ISO_ROOT=$WORK_DIR/iso-root
cd $ISO_ROOT

ISO_SQUASHFS=`find $ISO_ROOT -name "*.squashfs"`
if [ `wc -w <<< $ISO_SQUASHFS` -ne 1 ]; then
	echo "Unexpected number of '.squashfs' files: $ISO_SQUASHFS"
	exit 1
fi

UNSQUASHFS_DIR=$ISO_ROOT/squashfs-root
unsquashfs -d $UNSQUASHFS_DIR $ISO_SQUASHFS

cd $UNSQUASHFS_DIR

RAW_FILE=`find $UNSQUASHFS_DIR -name "*.raw"`
if [ `wc -w <<< $RAW_FILE` -ne 1 ]; then
	echo "Unexpected number of '.raw' files: $RAW_FILE"
	exit 1
fi

# Test the block size of the base image and adapt to suit either 512/4096 byte images
BLOCKSIZE=512
if ! guestfish -i --blocksize=$BLOCKSIZE -a $RAW_FILE $LUKSFLAG echo "[INFO] 512 byte sector check successful."; then
        echo "[WARN] Failed to access image with 512 byte sector size, trying 4096 bytes."
        BLOCKSIZE=4096
fi

virt-tar-out --blocksize=$BLOCKSIZE -a $RAW_FILE / - | pigz --best > $WORK_DIR/{{.ArchiveName}}
{{ else }}

# Test the block size of the base image and adapt to suit either 512/4096 byte images
BLOCKSIZE=512
if ! guestfish -i --blocksize=$BLOCKSIZE -a $IMG_PATH $LUKSFLAG echo "[INFO] 512 byte sector check successful."; then
        echo "[WARN] Failed to access image with 512 byte sector size, trying 4096 bytes."
        BLOCKSIZE=4096
fi

virt-tar-out --blocksize=$BLOCKSIZE -a $IMG_PATH $LUKSFLAG / - | pigz --best > $WORK_DIR/{{.ArchiveName}}

{{ end }}
