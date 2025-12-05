FROM golang:1-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY main.go cleanup.go ./
COPY pkg/ pkg/

# Build arguments for version information
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

# Build static binary with version info
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s -extldflags '-static' \
              -X 'main.Version=${VERSION}' \
              -X 'main.Commit=${COMMIT}' \
              -X 'main.BuildTime=${BUILD_TIME}'" \
    -o unifi-backup \
    .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /build/unifi-backup /unifi-backup

ENTRYPOINT ["/unifi-backup"]

CMD ["--config", "/config/config.yaml"]
