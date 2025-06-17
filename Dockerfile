ARG GOVERSION=latest
FROM golang:$GOVERSION AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 go build -o . -ldflags="-s -w -X main.release=$(git rev-parse HEAD)" ./cmd/vroom

FROM debian:bookworm-slim

EXPOSE 8080

RUN groupadd --gid 1000 vroom \
    && useradd -g vroom --uid 1000 vroom \
    && apt-get update \
    && apt-get install -y ca-certificates tzdata --no-install-recommends \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/vroom /bin/vroom

WORKDIR /var/vroom

ENTRYPOINT ["/bin/vroom"]
