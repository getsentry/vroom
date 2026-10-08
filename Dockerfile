ARG GOVERSION=latest
FROM golang:$GOVERSION AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 go build -o . -ldflags="-s -w -X main.release=$(git rev-parse HEAD)" ./cmd/vroom

# Distroless runtime: no shell or package manager. The base already ships the
# CA bundle (SSL_CERT_FILE), tzdata and a world-writable /tmp.
FROM us-docker.pkg.dev/sentryio/dhi-mirror/static:20250419-debian13 AS application-distroless

EXPOSE 8085

COPY --from=builder /src/vroom /bin/vroom

WORKDIR /var/vroom
# Same uid/gid as the debian image, so existing volumes and runAsUser keep working.
USER 1000:1000

ENTRYPOINT ["/bin/vroom"]

# Debian runtime. Keep it the last stage: builds without --target produce this image.
FROM debian:trixie-slim AS application

EXPOSE 8080

RUN groupadd --gid 1000 vroom \
    && useradd -g vroom --uid 1000 vroom \
    && apt-get update \
    && apt-get install -y ca-certificates tzdata --no-install-recommends \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/vroom /bin/vroom

WORKDIR /var/vroom
USER vroom

ENTRYPOINT ["/bin/vroom"]
