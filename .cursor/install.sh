#!/usr/bin/env bash
# Idempotent dependency refresh for the Budget App dev environment.
# Runs after the repository is checked out. Must terminate and be safe to re-run.
set -euo pipefail

cd "$(dirname "$0")/.."

# PostgreSQL is the app's database. Install the server package if it is missing.
# Package installation is idempotent; the running cluster is started in start.sh.
if ! command -v pg_ctlcluster >/dev/null 2>&1; then
  echo "[install] Installing PostgreSQL..."
  sudo apt-get update -qq
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq postgresql postgresql-contrib
else
  echo "[install] PostgreSQL already installed."
fi

# Download Go module dependencies. GOTOOLCHAIN=auto (the Go default) fetches the
# exact toolchain pinned in go.mod (go 1.26.x) on first use.
echo "[install] Downloading Go modules..."
go mod download

echo "[install] Done."
