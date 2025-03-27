ARG GOVERSION=latest

# Use Chainguard's minimal Go image for building
FROM cgr.dev/chainguard/go:$GOVERSION-dev AS builder

WORKDIR /src
COPY . .

# Build a statically linked binary
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o . -ldflags="-s -w -X main.release=$(git rev-parse HEAD)" ./cmd/vroom

# Create required directories
RUN mkdir -p /var/lib/sentry-profiles

# Use Chainguard's minimal go image for runtime
FROM cgr.dev/chainguard/static:latest

EXPOSE 8080

# Set environment variables
ENV SENTRY_BUCKET_PROFILES=file://localhost/var/lib/sentry-profiles

# Copy only the built binary from the builder stage
COPY --from=builder /src/vroom /bin/vroom

WORKDIR /var/vroom

# Run as non-root user (Chainguard images typically use 'nonroot')
USER nonroot:nonroot

ENTRYPOINT ["/bin/vroom"]
