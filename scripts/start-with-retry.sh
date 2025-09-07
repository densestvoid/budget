#!/bin/sh
set -e

echo "Starting Budget App with database connection retry logic..."

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
for i in $(seq 1 30); do
    if ./budget migrate status >/dev/null 2>&1; then
        echo "✅ PostgreSQL is ready!"
        break
    else
        echo "Waiting for PostgreSQL... (attempt $i/30)"
        sleep 2
    fi
done

# Run migrations
echo "Running database migrations..."
./budget migrate

# Start the application
echo "Starting Budget App server..."
exec ./budget serve