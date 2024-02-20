# ----- EIB Builder Image -----
FROM registry.suse.com/bci/golang:1.21

# Dependency uses by line
# 1. Podman Go library
RUN zypper install -y \
    gpgme-devel device-mapper-devel libbtrfs-devel

WORKDIR /src

COPY go.mod go.sum ./
COPY ./cmd ./cmd
COPY ./pkg ./pkg

RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    go build ./cmd/eib

# ----- Deliverable Image -----
FROM opensuse/leap:15.5

ENV HAULERVERSION 1.0.0
ENV NMCVERSION v0.2.0

RUN zypper addrepo https://download.opensuse.org/repositories/isv:SUSE:Edge:EdgeImageBuilder/SLE-15-SP5/isv:SUSE:Edge:EdgeImageBuilder.repo && \
    zypper --gpg-auto-import-keys refresh

# Dependency uses by line
# 1. ISO image building
# 2. RAW image modification on x86_64
# 3. Podman EIB library
# 4. RPM resolution logic
# 5. Network configurator
RUN zypper install -y \
    xorriso squashfs  \
    libguestfs kernel-default e2fsprogs parted gptfdisk btrfsprogs guestfs-tools lvm2 \
    podman \
    createrepo_c \
    helm \
    nm-configurator

# TODO: Install nmc via zypper once an RPM package is available
RUN curl -o /usr/local/bin/nmc -L https://github.com/suse-edge/nm-configurator/releases/download/${NMCVERSION}/nmc-linux-$(uname -m) && \
    chmod +x /usr/local/bin/nmc

# Hauler doesn't provide "uname -m" releases (x86_64/aarch64), but golang style ones (amd64/arm64) so we can use podman version instead
RUN mkdir -p /tmp/hauler && \
    curl -o /tmp/hauler/hauler.tar -L https://github.com/rancherfederal/hauler/releases/download/v${HAULERVERSION}/hauler_${HAULERVERSION}_$(podman version -f '{{.Client.OsArch}}' | sed 's!/!_!g').tar.gz && \
    tar -zxvf /tmp/hauler/hauler.tar -C /usr/local/bin hauler && \
    chmod a+x /usr/local/bin/hauler && \
    rm -Rf /tmp/hauler

RUN curl -o rke2_installer.sh -L https://get.rke2.io && \
    curl -o k3s_installer.sh -L https://get.k3s.io

COPY --from=0 /src/eib /bin/eib

CMD ["/bin/eib"]
