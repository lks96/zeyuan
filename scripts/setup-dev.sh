#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND="$ROOT/backend"
FRONTEND="$ROOT/frontend"
ENV_FILE="$BACKEND/.env"
ENV_EXAMPLE="$BACKEND/.env.example"

SKIP_MIGRATE="${SKIP_MIGRATE:-0}"

ensure_backend_env() {
  if [ ! -f "$ENV_FILE" ]; then
    cp "$ENV_EXAMPLE" "$ENV_FILE"
    echo "Created backend/.env. Please fill DB_PASSWORD before starting." >&2
    return 1
  fi

  if grep -Eq '^DB_PASSWORD[[:space:]]*=[[:space:]]*change_me[[:space:]]*$' "$ENV_FILE" && [ -z "${DB_PASSWORD:-}" ]; then
    echo "DB_PASSWORD in backend/.env is still change_me. Please set the real password." >&2
    return 1
  fi
}

echo "Checking Go / Node / npm..."
go version
node --version
npm --version

ENV_READY=1
ensure_backend_env || ENV_READY=0

echo "Installing frontend dependencies..."
(cd "$FRONTEND" && npm install)

echo "Downloading backend dependencies..."
(cd "$BACKEND" && go mod download)

if [ "$ENV_READY" = "1" ] && [ "$SKIP_MIGRATE" != "1" ]; then
  echo "Preparing database schema..."
  (cd "$BACKEND" && go run ./cmd/dbprepare)
fi

if [ "$ENV_READY" != "1" ]; then
  echo
  echo "Setup is incomplete. Edit backend/.env, then run bash ./scripts/setup-dev.sh again."
  exit 1
fi

echo
echo "Setup complete. Start the app with: bash ./scripts/start-dev.sh"
