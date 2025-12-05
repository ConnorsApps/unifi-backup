FROM golang:1-bookworm AS builder

RUN apt-get update && \
    apt-get install -y --no-install-recommends git ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY main.go cleanup.go utils.go ./
COPY pkg/ pkg/

# Build arguments for version information
ARG VERSION=dev
ARG COMMIT=unknown

# Build static binary with version info
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s -extldflags '-static' \
              -X 'main.Version=${VERSION}' \
              -X 'main.Commit=${COMMIT}'" \
    -o unifi-backup \
    .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /build/unifi-backup /unifi-backup

ENTRYPOINT ["/unifi-backup"]

CMD ["--config", "/config/config.yaml"]
