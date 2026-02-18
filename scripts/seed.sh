#!/usr/bin/env bash
# seed.sh — Apply the EdgeClaw schema and load demo sensor data.
#
# Connects to the local TimescaleDB instance (defaults match docker-compose.yml)
# and runs:
#   1. deploy/schema.sql  — creates the sensors hypertable (idempotent)
#   2. scripts/seed.sql   — inserts ~100 rows of realistic demo data
#
# Usage:
#   ./scripts/seed.sh
#
# Environment variables (all optional — defaults match Docker Compose):
#   PGHOST      PostgreSQL host          (default: localhost)
#   PGPORT      PostgreSQL port          (default: 5432)
#   PGUSER      PostgreSQL user          (default: edgeclaw)
#   PGPASSWORD  PostgreSQL password      (default: changeme)
#   PGDATABASE  PostgreSQL database      (default: edgeclaw)
#
# Examples:
#   # Use all defaults (local Docker Compose stack)
#   ./scripts/seed.sh
#
#   # Override host and password for a remote node
#   PGHOST=192.168.1.50 PGPASSWORD=secret ./scripts/seed.sh

set -euo pipefail

# ---------------------------------------------------------------------------
# Resolve paths relative to the repository root
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SCHEMA_FILE="$REPO_ROOT/deploy/schema.sql"
SEED_FILE="$REPO_ROOT/scripts/seed.sql"

# ---------------------------------------------------------------------------
# Database connection defaults (match deploy/docker-compose.yml)
# ---------------------------------------------------------------------------
export PGHOST="${PGHOST:-localhost}"
export PGPORT="${PGPORT:-5432}"
export PGUSER="${PGUSER:-edgeclaw}"
export PGPASSWORD="${PGPASSWORD:-changeme}"
export PGDATABASE="${PGDATABASE:-edgeclaw}"

# ---------------------------------------------------------------------------
# Preflight checks
# ---------------------------------------------------------------------------
if ! command -v psql &>/dev/null; then
    echo "ERROR: psql not found. Install postgresql-client or add it to PATH." >&2
    exit 1
fi

if [ ! -f "$SCHEMA_FILE" ]; then
    echo "ERROR: Schema file not found: $SCHEMA_FILE" >&2
    exit 1
fi

if [ ! -f "$SEED_FILE" ]; then
    echo "ERROR: Seed file not found: $SEED_FILE" >&2
    exit 1
fi

echo "EdgeClaw seed script"
echo "  Host:     $PGHOST:$PGPORT"
echo "  Database: $PGDATABASE"
echo "  User:     $PGUSER"
echo ""

# ---------------------------------------------------------------------------
# 1. Apply schema (idempotent — uses IF NOT EXISTS throughout)
# ---------------------------------------------------------------------------
echo "[1/2] Applying schema from $SCHEMA_FILE ..."
psql -v ON_ERROR_STOP=1 -f "$SCHEMA_FILE"
echo "  Schema applied."
echo ""

# ---------------------------------------------------------------------------
# 2. Load seed data
# ---------------------------------------------------------------------------
echo "[2/2] Loading seed data from $SEED_FILE ..."
psql -v ON_ERROR_STOP=1 -f "$SEED_FILE"
echo ""
echo "Seed complete. The summary above shows the data that was inserted."
