# ----- EIB Builder Image -----
FROM golang:1.21

WORKDIR /src
COPY . ./
RUN go build ./cmd/eib


# ----- Deliverable Image -----
FROM registry.suse.com/bci/bci-base:15.4

COPY --from=0 /src/eib /bin/eib

CMD ["/bin/eib"]
