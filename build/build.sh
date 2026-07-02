#!/usr/bin/env bash
set -euo pipefail

# This script builds and runs the frontend and backend services using Docker Compose.
# Docker Compose is the standard tool for managing multi-container Docker applications,
# simplifying networking, service dependencies, and startup/shutdown procedures.

# Ensure Docker Compose is available
if ! command -v docker-compose &> /dev/null
then
    echo "Error: docker-compose could not be found. Please install it to continue."
    exit 1
fi

# The docker-compose.yml file is expected to be in the same directory as this script.
COMPOSE_FILE="$(dirname "$0")/docker-compose.yml"

if [ ! -f "$COMPOSE_FILE" ]; then
    echo "Error: docker-compose.yml not found in $(dirname "$0")/"
    exit 1
fi

echo "=========================================="
echo "Building and running Alert Manager services..."
echo "=========================================="
echo "Using Compose file: $COMPOSE_FILE"
echo ""

# Stop any running services defined in the compose file to avoid conflicts
echo "Stopping existing services (if any)..."
docker-compose -f "$COMPOSE_FILE" down

# Build the images defined in the compose file
echo ""
echo "Building Docker images for backend and frontend..."
docker-compose -f "$COMPOSE_FILE" build

# Start all services in detached mode
echo ""
echo "Starting all services in the background..."
docker-compose -f "$COMPOSE_FILE" up -d

echo ""
echo "=========================================="
echo "✓ Services are up and running!"
echo ""
echo "  - Frontend is available at: http://localhost:8081"
echo "  - Backend API is available at: http://localhost:8080"
echo "  - PostgreSQL DB is available at: localhost:5432"
echo ""
echo "To view logs, run: docker-compose -f '$COMPOSE_FILE' logs -f"
echo "To stop services, run: docker-compose -f '$COMPOSE_FILE' down"
echo "=========================================="
