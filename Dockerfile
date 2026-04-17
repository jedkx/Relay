# Build relay + relay-mock (static binaries, no CGO).
FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/relay ./cmd/relay
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/relay-mock ./cmd/relay-mock

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /out/relay /app/relay
COPY --from=builder /out/relay-mock /app/relay-mock

# Default: relay API (:8080). Override command for mock (:8081).
EXPOSE 8080

CMD ["/app/relay"]
