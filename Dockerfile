# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o btrfs-csi-plugin main.go

# Runtime stage
FROM alpine:3.22

# Install runtime dependencies
RUN apk update && \
    apk add --no-cache \
    util-linux \
    ca-certificates

# Create necessary directories
RUN mkdir -p /var/lib/btrfs-csi

# Copy binary from builder stage
COPY --from=builder /app/btrfs-csi-plugin /usr/bin/btrfs-csi-plugin

# Set working directory
WORKDIR /var/lib/btrfs-csi

# Run the driver
ENTRYPOINT ["/usr/bin/btrfs-csi-plugin"]
