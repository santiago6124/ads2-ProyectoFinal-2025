# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Initialize minimal go module
RUN go mod init wallet-api

# Copy only cmd/server for simplified build
COPY cmd/server/main.go ./cmd/server/

# Download dependencies needed for main (use older gin compatible with Go 1.21)
RUN go get github.com/gin-gonic/gin@v1.9.1 && go mod tidy

# Build the application (simple version)
RUN CGO_ENABLED=0 GOOS=linux go build -o wallet-api ./cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and wget for healthcheck
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/wallet-api .

# Create necessary directories
RUN mkdir -p /app/logs && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./wallet-api"]