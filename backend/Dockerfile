# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install deps
RUN apk add --no-cache git ca-certificates tzdata

# Go env
ENV GOPROXY=https://goproxy.io,direct \
    CGO_ENABLED=0 \
    GOOS=linux

# Better layer cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api


# Runtime stage
FROM alpine:3.18

WORKDIR /app

# NOTE: you use wget in HEALTHCHECK, so must install it
RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /out/api ./api

# Create non-root user
RUN adduser -D -s /bin/sh appuser
USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://localhost:8080/health || exit 1

CMD ["./api"]
