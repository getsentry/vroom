ARG GOVERSION=latest
FROM golang:$GOVERSION AS builder

WORKDIR /src
COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o . -ldflags="-s -w" ./cmd/vroom

FROM alpine

EXPOSE 8080

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /src/vroom /bin/vroom

WORKDIR /var/vroom

ENTRYPOINT ["/bin/vroom"]
