# Build stage
FROM golang:1.21-alpine AS builder

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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o btrfs-csi main.go

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    btrfs-progs \
    util-linux \
    ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh btrfs-csi

# Create necessary directories
RUN mkdir -p /var/lib/btrfs-csi && \
    chown btrfs-csi:btrfs-csi /var/lib/btrfs-csi

# Copy binary from builder stage
COPY --from=builder /app/btrfs-csi /bin/btrfs-csi

# Set ownership and permissions
RUN chown btrfs-csi:btrfs-csi /bin/btrfs-csi && \
    chmod +x /bin/btrfs-csi

# Switch to non-root user
USER btrfs-csi

# Set working directory
WORKDIR /var/lib/btrfs-csi

# Expose CSI socket
VOLUME ["/var/lib/btrfs-csi"]

# Run the driver
ENTRYPOINT ["/bin/btrfs-csi"]