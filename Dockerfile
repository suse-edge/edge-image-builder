# ----- EIB Builder Image -----
FROM registry.suse.com/bci/golang:1.21

WORKDIR /src
COPY . ./

# Dependency uses by line
# 1. Podman Go library
RUN zypper install -y \
    gpgme-devel device-mapper-devel libbtrfs-devel

RUN go build ./cmd/eib


# ----- Deliverable Image -----
FROM opensuse/leap:15.5

# Dependency uses by line
# 1. ISO image building
# 2. RAW image modification on x86_64
# 3. Podman EIB library
# 4. RPM resolution logic
RUN zypper install -y \
    xorriso squashfs  \
    libguestfs kernel-default e2fsprogs parted gptfdisk btrfsprogs \
    podman \
    createrepo_c

# TODO: Install nmc via zypper once an RPM package is available
RUN curl -o nmc-aarch64 -L https://github.com/suse-edge/nm-configurator/releases/download/v0.2.0/nmc-linux-aarch64 && \
    chmod +x nmc-aarch64 && \
    curl -o nmc-x86_64 -L https://github.com/suse-edge/nm-configurator/releases/download/v0.2.0/nmc-linux-x86_64 && \
    chmod +x nmc-x86_64 && \
    cp nmc-$(uname -m) /usr/local/bin/nmc

RUN curl -o rke2_installer.sh -L https://get.rke2.io

COPY --from=0 /src/eib /bin/eib

CMD ["/bin/eib"]
