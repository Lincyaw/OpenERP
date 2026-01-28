# Build stage
FROM golang:1.25.5-trixie AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/server ./cmd/server

# Runtime stage
FROM debian:trixie-slim

# Change to China mirrors
RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources || \
    sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list || true && \
    sed -i 's/security.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list || true

# Install runtime dependencies including Chromium and Chinese fonts
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    # Chromium for PDF generation (headless)
    chromium \
    # Chinese fonts for PDF generation
    fonts-noto-cjk \
    fontconfig \
    fonts-dejavu \
    wget \
    && rm -rf /var/lib/apt/lists/* \
    && fc-cache -f -v

# Set Chromium environment variables for chromedp
ENV CHROME_BIN=/usr/bin/chromium
ENV CHROMEDP_NO_SANDBOX=true

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/bin/server /app/server

# Copy migrations
COPY --from=builder /app/migrations /app/migrations

# Create data directory for PDF storage
RUN mkdir -p /data/prints

# Expose port
EXPOSE 8080

# Health check (use -q -O /dev/null instead of --spider to send GET request)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/health || exit 1

# Run the application
CMD ["/app/server"]
