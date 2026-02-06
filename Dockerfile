# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for cross-compilation
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

# Build the binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /app/bin/termdash \
    ./cmd/termdash

# Final stage - minimal image for running
FROM alpine:3.19

# Add ca-certificates for HTTPS and tzdata for timezone
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 termdash

# Copy binary from builder
COPY --from=builder /app/bin/termdash /usr/local/bin/termdash

# Use non-root user
USER termdash

# Set terminal environment
ENV TERM=xterm-256color

ENTRYPOINT ["termdash"]
