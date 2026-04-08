FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY go.mod ./
RUN go mod download 2>/dev/null || true
COPY cmd/ cmd/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /mana-health ./cmd/mana-health/ && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /mana-proxy ./cmd/mana-proxy/ && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /mana-agent ./cmd/mana-agent/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /mana-health /usr/local/bin/
COPY --from=builder /mana-proxy /usr/local/bin/
COPY --from=builder /mana-agent /usr/local/bin/
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 8080 8081 8082
ENTRYPOINT ["/entrypoint.sh"]
