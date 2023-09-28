FROM registry.suse.com/bci/bci-base:15.4

# Prepare the OS with dependencies
RUN zypper install -y \
    go \
    mkisofs

# Establish the working directory
WORKDIR /eib

# Build EIB
COPY . ./
RUN go build ./cmd/eib

# Run the builder
CMD ["./eib"]
