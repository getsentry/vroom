ARG GOVERSION=latest
FROM golang:$GOVERSION AS builder

WORKDIR /src
COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o . -ldflags="-s -w -X main.release=$(git rev-parse HEAD)" ./cmd/vroom

FROM debian:bullseye-slim

EXPOSE 8080

ARG PROFILES_DIR=/var/lib/sentry-profiles

RUN apt-get install ca-certificates tzdata && mkdir -p $PROFILES_DIR && \
    rm -r /var/lib/apt/lists/*

ENV SENTRY_BUCKET_PROFILES=file://localhost/$PROFILES_DIR

COPY --from=builder /src/vroom /bin/vroom

WORKDIR /var/vroom

ENTRYPOINT ["/bin/vroom"]
