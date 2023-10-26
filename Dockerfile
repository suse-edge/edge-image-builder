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
    xorriso  \
    libguestfs kernel-default e2fsprogs parted gptfdisk btrfsprogs

COPY --from=0 /src/eib /bin/eib

CMD ["/bin/eib"]
