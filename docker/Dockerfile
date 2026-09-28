# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/api ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl \
    && adduser -D -u 10001 appuser

COPY --from=builder /out/api /usr/local/bin/api

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=5 \
  CMD curl -fsS http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/api"]
