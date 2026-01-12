# ----- EIB Builder Image -----
FROM registry.suse.com/bci/golang:1.25.5-1.78.3

# Dependency uses by line
# 1. Podman Go library
RUN zypper install -y \
    gpgme-devel device-mapper-devel libbtrfs-devel

WORKDIR /src

COPY go.mod go.sum ./
COPY ./cmd ./cmd
COPY ./pkg ./pkg
COPY .git .git

RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build ./cmd/eib

# ----- Deliverable Image -----
FROM opensuse/leap:15.6

# Dependency uses by line
# 1. ISO image building
# 2. RAW image modification on x86_64 and aarch64
# 3. Podman EIB library
# 4. RPM resolution logic
# 5. Embedded artefact registry
# 6. Network configuration
# 7. SUSE registry certificates
RUN zypper addrepo https://download.opensuse.org/repositories/isv:/SUSE:/Edge:/Factory/standard/isv:SUSE:Edge:Factory.repo && \
    zypper addrepo https://download.opensuse.org/repositories/SUSE:CA/15.6/SUSE:CA.repo && \
    zypper --gpg-auto-import-keys refresh && \
    zypper install -y \
    xorriso squashfs  \
    libguestfs kernel-default e2fsprogs parted gptfdisk btrfsprogs guestfs-tools lvm2 qemu-uefi-aarch64 \
    podman \
    createrepo_c \
    helm hauler \
    nm-configurator \
    ca-certificates-suse && \
    zypper clean -a

# Make adjustments for running guestfish and image modifications on aarch64
# guestfish looks for very specific locations on the filesystem for UEFI firmware
# and also expects the boot kernel to be a portable executable (PE), not ELF.
RUN mkdir -p /usr/share/edk2/aarch64 && \
	cp /usr/share/qemu/aavmf-aarch64-code.bin /usr/share/edk2/aarch64/QEMU_EFI-pflash.raw && \
	cp /usr/share/qemu/aavmf-aarch64-vars.bin /usr/share/edk2/aarch64/vars-template-pflash.raw && \
	mv /boot/vmlinux* /boot/backup-vmlinux

COPY --from=0 /src/eib /bin/eib
COPY config/artifacts.yaml artifacts.yaml

# Test eib executable to verify glibc compatibility on openSUSE Leap
RUN /bin/eib version

ENTRYPOINT ["/bin/eib"]
