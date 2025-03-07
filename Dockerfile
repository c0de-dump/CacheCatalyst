FROM golang:1.22-alpine AS builder

COPY ./caddy .
RUN go mod download
RUN go build -o /caddy ./cmd/caddy

FROM alpine:latest

WORKDIR /app
COPY --from=builder /caddy newcaddy

# ENTRYPOINT [ "./newcaddy", "run", "--config", "/tmp/storage/Caddyfile" ]
ENTRYPOINT [ "./newcaddy", "run" ]
