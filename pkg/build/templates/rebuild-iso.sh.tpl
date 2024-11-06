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
sed -i '/root=install:CDLABEL=INSTALL/ s|$| rd.kiwi.oem.installdevice={{.InstallDevice}} |' ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg
{{ end -}}

{{ if (gt (len .KernelArgs) 0) -}}
# Remove all original kernel arguments from ISO command line that match input kernelArgs *and* have values
{{ range .KernelArgsList -}}
value=$(echo {{ . }} | cut -f1 -d"=")
sed -i "s/$value=[^=]//" ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg
{{ end -}}

# Unpack the initrd from the SelfInstall ISO and copy the early microcode into new initrd
mkdir -p ${ISO_EXTRACT_DIR}/temp-initram/early/ ${ISO_EXTRACT_DIR}/temp-initram/main/
cp ${ISO_EXTRACT_DIR}/boot/{{ .Arch }}/loader/initrd ${ISO_EXTRACT_DIR}/temp-initram/
cd ${ISO_EXTRACT_DIR}/temp-initram/early && lsinitrd --unpackearly ${ISO_EXTRACT_DIR}/temp-initram/initrd
find . -print0 | cpio --null --create --format=newc > ${ISO_EXTRACT_DIR}/temp-initram/new-initrd
# NOTE: We pipe the following command to true to avoid issues with mknod failing when unprivileged
cd ${ISO_EXTRACT_DIR}/temp-initram/main && lsinitrd --unpack ${ISO_EXTRACT_DIR}/temp-initram/initrd || true

# Remove the original kernel arguments from initrd config that match input kernelArgs and add desired ones
{{ range .KernelArgsList -}}
value=$(echo {{ . }} | cut -f1 -d"=")
sed -i "s/$value=[^=]//" config.bootoptions
{{ end -}}
sed -i '1s|$| {{ .KernelArgs }}|' config.bootoptions

# Repack the contents of the initrd into the new file, including the new kernel cmdline arguments
find . | cpio --create --format=newc >> ${ISO_EXTRACT_DIR}/temp-initram/new-initrd

# Add the desired kernel cmdline arguments to the ISO kernel cmdline so they're available during deployment
sed -i '/root=install:CDLABEL=INSTALL/ s|$| {{.KernelArgs}} |' ${ISO_EXTRACT_DIR}/boot/grub2/grub.cfg
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
{{- if (gt (len .KernelArgs) 0) }}
        -map ${ISO_EXTRACT_DIR}/temp-initram/new-initrd /boot/{{ .Arch }}/loader/initrd \
{{- end }}
        -boot_image any replay -changes_pending yes
