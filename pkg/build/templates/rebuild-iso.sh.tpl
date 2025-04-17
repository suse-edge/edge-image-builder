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

MD5_CHECKSUM_FILE=`find "${RAW_EXTRACT_DIR}" -name "*.md5"`
SHA256_CHECKSUM_FILE=`find "${RAW_EXTRACT_DIR}" -name "*.sha256"`

if [[ -n "$SHA256_CHECKSUM_FILE" ]]; then
  CHECKSUM_TYPE="sha256"
  CHECKSUM_FILE="$SHA256_CHECKSUM_FILE"
elif [[ -n "$MD5_CHECKSUM_FILE" ]]; then
  CHECKSUM_TYPE="md5"
  CHECKSUM_FILE="$MD5_CHECKSUM_FILE"
else
  echo "Error: No MD5 or SHA256 checksum file found in ${RAW_EXTRACT_DIR}" >&2
  exit 1
fi

BLK_CONF=$(awk '{print $2 " " $3;}' "$CHECKSUM_FILE")

if [[ "$CHECKSUM_TYPE" == "md5" ]]; then
  echo "$(md5sum "${RAW_IMAGE_FILE}" | awk '{print $1;}') $BLK_CONF" > "$CHECKSUM_FILE"
elif [[ "$CHECKSUM_TYPE" == "sha256" ]]; then
  echo "$(sha256sum "${RAW_IMAGE_FILE}" | awk '{print $1;}') $BLK_CONF" > "$CHECKSUM_FILE"
fi

# Resquash the raw image
SQUASH_IMAGE_FILE=`find ${ISO_EXTRACT_DIR} -name "*.squashfs"`
SQUASH_BASENAME=`basename ${SQUASH_IMAGE_FILE}`
NEW_SQUASH_FILE=${RAW_EXTRACT_DIR}/${SQUASH_BASENAME}

# Select the desired install device - assumes data destruction and makes the installation fully unattended by enabling GRUB timeout
{{ if ne .InstallDevice "" -}}
echo -e "set timeout=3\nset timeout_style=menu\n$(cat ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg)" > ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg
sed -i '/root=install:CDLABEL=INSTALL/ s|$| rd.kiwi.oem.installdevice={{.InstallDevice}} |' ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg
{{ end -}}

# Ensure that kernel arguments are appended to ISO grub.cfg so they are applied to firstboot via kexec
{{ if (gt (len .KernelArgs) 0) -}}
sed -i '/root=install:CDLABEL=INSTALL/ s|$| rd.kiwi.install.pass.bootparam {{.KernelArgs}} |' ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg
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
