#!/bin/sh
set -e

echo "Starting Budget App server..."
echo "Environment check:"
echo "BUDGET_DATABASE_URL: ${BUDGET_DATABASE_URL}"
echo "BUDGET_PORT: ${BUDGET_PORT}"
echo "BUDGET_ENV: ${BUDGET_ENV}"

# Note: Database migrations are handled by the pre-deploy job
# This container only needs to start the web server

# Start the application directly
exec ./budget serve