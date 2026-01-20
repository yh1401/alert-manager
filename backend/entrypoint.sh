#!/bin/bash
set -e

# Default credentials - can be overridden by docker-compose environment variables
DEFAULT_ADMIN_USER="admin"
DEFAULT_ADMIN_PASSWORD="password123"

# Use environment variables if they are set, otherwise use defaults
ADMIN_USER=${ADMIN_USER:-$DEFAULT_ADMIN_USER}
ADMIN_PASSWORD=${ADMIN_PASSWORD:-$DEFAULT_ADMIN_PASSWORD}

# This function runs in the background to set up the initial admin user.
# It's idempotent, handled by the Go program.
setup_admin() {
    # Give the backend service and DB a moment to initialize
    echo "Admin setup process will start in 10 seconds..."
    sleep 10

    echo "Running admin setup..."
    # The setup-admin tool is idempotent. It will create the admin if it doesn't
    # exist, update the role if needed, or do nothing if already configured.
    /app/setup-admin "$ADMIN_USER" "$ADMIN_PASSWORD"
    echo "✓ Admin setup process complete."
}

# Run setup in the background
setup_admin &

# Execute the main application
exec /app/alert-manager-backend
