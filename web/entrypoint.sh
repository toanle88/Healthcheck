#!/bin/sh
# This script dynamically generates the env.js file at runtime
# based on the environment variables passed to the Docker container.

# Path to the env.js file served by Nginx
ENV_FILE="/usr/share/nginx/html/env.js"

echo "Generating $ENV_FILE..."

# Write the window.ENV object
echo "window.ENV = {" > $ENV_FILE
echo "  VITE_API_URL: \"$VITE_API_URL\"," >> $ENV_FILE
echo "  VITE_APP_VERSION: \"$VITE_APP_VERSION\"," >> $ENV_FILE
echo "  VITE_ENTRA_CLIENT_ID: \"$VITE_ENTRA_CLIENT_ID\"," >> $ENV_FILE
echo "  VITE_ENTRA_TENANT_ID: \"$VITE_ENTRA_TENANT_ID\"" >> $ENV_FILE
echo "};" >> $ENV_FILE

echo "Environment variables injected successfully."

# Execute the main process (Nginx)
# "exec" ensures Nginx replaces this shell script as PID 1,
# which is important for proper signal handling in Docker.
exec "$@"
