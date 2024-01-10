#!/bin/bash
set -euo pipefail

#  Template Fields
#  IsoExtractDir - Full path to the directory where the ISO was extracted
#  RawExtractDir - Full path to the directory where the RAW image was extracted
#  IsoSource - Full path to the original ISO that was extracted
#  OutputImageFilename - Full path and name of the ISO to create
#  CombustionDir - Full path to the combustion directory to include in the new ISO

ISO_EXTRACT_DIR={{.IsoExtractDir}}
RAW_EXTRACT_DIR={{.RawExtractDir}}
ISO_SOURCE={{.IsoSource}}
OUTPUT_IMAGE={{.OutputImageFilename}}
COMBUSTION_DIR={{.CombustionDir}}

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

cd ${RAW_EXTRACT_DIR}

echo "Squash"
mksquashfs ${RAW_IMAGE_FILE} ${CHECKSUM_FILE} ${NEW_SQUASH_FILE}

# Rebuild the previously extracted ISO with the new squashed raw image
xorriso -indev ${ISO_SOURCE} \
        -outdev ${OUTPUT_IMAGE} \
        -map ${NEW_SQUASH_FILE} /${SQUASH_BASENAME} \
        -map ${COMBUSTION_DIR} /combustion \
        -boot_image any replay -changes_pending yes
