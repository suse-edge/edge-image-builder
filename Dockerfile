# ----- EIB Builder Image -----
FROM registry.suse.com/bci/golang:1.21

WORKDIR /src
COPY . ./
RUN go build ./cmd/eib


# ----- Deliverable Image -----
FROM opensuse/leap:15.5

# Dependency uses by line
# 1. ISO image building
# 2. RAW image modification on x86_64
RUN zypper install -y \
    xorriso squashfs  \
    libguestfs kernel-default e2fsprogs parted gptfdisk btrfsprogs

# TODO: Install nmc via zypper once an RPM package is available
RUN curl -o nmc-aarch64 -L https://github.com/suse-edge/nm-configurator/releases/download/v0.2.0/nmc-linux-aarch64 && \
    chmod +x nmc-aarch64 && \
    curl -o nmc-x86_64 -L https://github.com/suse-edge/nm-configurator/releases/download/v0.2.0/nmc-linux-x86_64 && \
    chmod +x nmc-x86_64 && \
    cp nmc-$(uname -m) /usr/local/bin/nmc

RUN curl -o hauler-amd64.tar -L https://github.com/rancherfederal/hauler/releases/download/v0.4.2/hauler_0.4.2_linux_amd64.tar.gz && \
    tar -xf hauler-amd64.tar && \
    mv hauler /usr/bin/hauler-amd64 && \
    curl -o hauler-arm64.tar -L https://github.com/rancherfederal/hauler/releases/download/v0.4.2/hauler_0.4.2_linux_arm64.tar.gz && \
    tar -xf hauler-arm64.tar && \
    mv hauler /usr/bin/hauler-arm64

COPY --from=0 /src/eib /bin/eib

CMD ["/bin/eib"]
