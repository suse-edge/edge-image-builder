#!/bin/bash
set -euo pipefail

#  Template Fields
#  IsoExtractDir - Full path to the directory (under the build directory) where the ISO should be extracted
#  RawExtractDir - Full path to the directory (under the build directory) where the RAW image should be
#                  unsquashed
#  IsoSource - Full path to the ISO to extract

ISO_EXTRACT_DIR={{.IsoExtractDir}}
RAW_EXTRACT_DIR={{.RawExtractDir}}
ISO_SOURCE={{.IsoSource}}

# Create the extract directories
mkdir -p ${ISO_EXTRACT_DIR}

# Extract the contents of the ISO to the build directory
xorriso -osirrox on -indev ${ISO_SOURCE} extract / ${ISO_EXTRACT_DIR}

# Unsquash the raw image
SQUASHED_IMAGE_NAME=`find ${ISO_EXTRACT_DIR} -name "*.squashfs"`
unsquashfs -d ${RAW_EXTRACT_DIR} ${SQUASHED_IMAGE_NAME}
