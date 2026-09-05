#!/usr/bin/env bash
# Per-boot runtime reconciliation for the Budget App dev environment.
# Starts PostgreSQL, ensures the role/database exist, and applies migrations.
# Must tolerate restarts and return once the database is ready.
set -euo pipefail

cd "$(dirname "$0")/.."

PGVER="$(ls /usr/lib/postgresql 2>/dev/null | sort -V | tail -1)"
if [ -z "${PGVER:-}" ]; then
  echo "[start] PostgreSQL is not installed; run install first." >&2
  exit 1
fi

# Start the default cluster only if it is not already online (idempotent).
if ! pg_lsclusters -h 2>/dev/null | awk '{print $4}' | grep -q '^online$'; then
  echo "[start] Starting PostgreSQL ${PGVER}/main..."
  sudo pg_ctlcluster "$PGVER" main start
else
  echo "[start] PostgreSQL already online."
fi

# Wait for the server to accept connections.
for _ in $(seq 1 30); do
  if pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
pg_isready -h localhost -p 5432 >/dev/null 2>&1 || { echo "[start] PostgreSQL did not become ready." >&2; exit 1; }

# Ensure the credentials and database the app expects exist (idempotent).
# Matches the default DSN: postgres://postgres:password@localhost:5432/budget
sudo -u postgres psql -v ON_ERROR_STOP=1 -tAc "ALTER USER postgres PASSWORD 'password';" >/dev/null
if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='budget'" | grep -q 1; then
  echo "[start] Creating 'budget' database..."
  sudo -u postgres createdb budget
fi

# Apply embedded Goose migrations. Goose tracks applied versions, so this is safe
# to run on every boot.
echo "[start] Applying database migrations..."
go run main.go migrate

echo "[start] Ready."
