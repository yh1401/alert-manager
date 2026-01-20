# Stage 1: Build the Go application
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go app and admin tool
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o /alert-manager-backend ./main.go && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o /setup-admin ./cmd/setup_admin/main.go

# Stage 2: Create the final lightweight image
FROM debian:bullseye-slim

# Install prometheus for promtool and ca-certificates for HTTPS calls
RUN apt-get update && \
    apt-get install -y prometheus ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the built binaries from the builder stage
COPY --from=builder /alert-manager-backend /app/alert-manager-backend
COPY --from=builder /setup-admin /app/setup-admin

# Copy the static agent binary and the config file
COPY static /app/static
COPY config/config.yaml /app/config/config.yaml

# Copy and set up the entrypoint script
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Expose the port the app runs on
EXPOSE 8080

# The DB_DSN environment variable can be used to override the DSN in config.yaml
# Example: -e "DB_DSN=host=postgres user=postgres password=secret dbname=alert_manager port=5432 sslmode=disable"
ENV PROMTOOL_PATH=/usr/bin/promtool

# Run the entrypoint script
ENTRYPOINT ["/app/entrypoint.sh"]
