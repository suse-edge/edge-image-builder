#!/bin/bash
set -euo pipefail

#  Template Fields
#  IsoExtractDir - Full path to the directory where the ISO was extracted
#  RawExtractDir - Full path to the directory where the RAW image was extracted
#  IsoSource - Full path to the original ISO that was extracted
#  OutputImageFilename - Full path and name of the ISO to create
#  CombustionDir - Full path to the combustion directory to include in the new ISO
#  ArtefactsDir - Full path to the artefacts directory to include in the new ISO

ISO_EXTRACT_DIR={{.IsoExtractDir}}
RAW_EXTRACT_DIR={{.RawExtractDir}}
ISO_SOURCE={{.IsoSource}}
OUTPUT_IMAGE={{.OutputImageFilename}}
COMBUSTION_DIR={{.CombustionDir}}
ARTEFACTS_DIR={{.ArtefactsDir}}

cd ${ISO_EXTRACT_DIR}

# Regenerate the checksum, overwriting the existing one that was unsquashed
RAW_IMAGE_FILE=`find ${RAW_EXTRACT_DIR} -name "*.raw"`
CHECKSUM_FILE=`find ${RAW_EXTRACT_DIR} -name "*.md5"`
BLK_CONF=$(awk '{print $2 " " $3;}' $CHECKSUM_FILE)
echo "$(md5sum ${RAW_IMAGE_FILE} | awk '{print $1;}') $BLK_CONF" > ${CHECKSUM_FILE}

# Resquash the raw image
SQUASH_IMAGE_FILE=`find ${ISO_EXTRACT_DIR} -name "*.squashfs"`
SQUASH_BASENAME=`basename ${SQUASH_IMAGE_FILE}`
NEW_SQUASH_FILE=${RAW_EXTRACT_DIR}/${SQUASH_BASENAME}

# Select the desired install device - assumes data destruction and makes the installation fully unattended by enabling GRUB timeout
{{ if ne .InstallDevice "" -}}
echo -e "set timeout=3\nset timeout_style=menu\n$(cat ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg)" > ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg
sed -i '/ignition.platform/ s|$| rd.kiwi.oem.installdevice={{.InstallDevice}} |' ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg
{{ end -}}


cd ${RAW_EXTRACT_DIR}
mksquashfs ${RAW_IMAGE_FILE} ${CHECKSUM_FILE} ${NEW_SQUASH_FILE}

# Rebuild the previously extracted ISO with the new squashed raw image
xorriso -indev ${ISO_SOURCE} \
        -outdev ${OUTPUT_IMAGE} \
        -map ${NEW_SQUASH_FILE} /${SQUASH_BASENAME} \
        -map ${COMBUSTION_DIR} /combustion \
        -map ${ARTEFACTS_DIR} /artefacts \
{{- if .InstallDevice }}
        -map ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg /boot/grub2/grub.cfg \
{{- end }}
        -boot_image any replay -changes_pending yes
