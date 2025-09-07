#!/bin/sh
set -e

echo "Starting Budget App with database connection retry logic..."
echo "Environment check:"
echo "BUDGET_DATABASE_URL: ${BUDGET_DATABASE_URL}"
echo "BUDGET_PORT: ${BUDGET_PORT}"
echo "All BUDGET_ variables:"
env | grep BUDGET || echo "No BUDGET_ variables found"
echo "All environment variables:"
env

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
for i in $(seq 1 30); do
    echo "Testing database connection (attempt $i/30)..."
    if ./budget migrate status >/dev/null 2>&1; then
        echo "✅ PostgreSQL is ready!"
        break
    else
        echo "PostgreSQL not ready yet... (attempt $i/30)"
        if [ $i -eq 30 ]; then
            echo "❌ PostgreSQL failed to become ready after 60 seconds"
            echo "Current environment:"
            env | grep BUDGET
            exit 1
        fi
        sleep 2
    fi
done

# Run migrations
echo "Running database migrations..."
if ./budget migrate; then
    echo "✅ Database migrations completed successfully"
else
    echo "❌ Database migrations failed"
    exit 1
fi

# Start the application
echo "Starting Budget App server..."
exec ./budget serve