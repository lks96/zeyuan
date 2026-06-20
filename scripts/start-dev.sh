#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND="$ROOT/backend"
FRONTEND="$ROOT/frontend"
ENV_FILE="$BACKEND/.env"
ENV_EXAMPLE="$BACKEND/.env.example"

SKIP_INSTALL="${SKIP_INSTALL:-0}"
SKIP_MIGRATE="${SKIP_MIGRATE:-0}"
BACKEND_PID=""
FRONTEND_PID=""

cleanup() {
  if [ -n "$BACKEND_PID" ] && kill -0 "$BACKEND_PID" 2>/dev/null; then
    kill "$BACKEND_PID" 2>/dev/null || true
  fi
  if [ -n "$FRONTEND_PID" ] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
    kill "$FRONTEND_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

if [ ! -f "$ENV_FILE" ]; then
  cp "$ENV_EXAMPLE" "$ENV_FILE"
  echo "Created backend/.env. Please fill DB_PASSWORD before starting." >&2
  exit 1
fi

if grep -Eq '^DB_PASSWORD[[:space:]]*=[[:space:]]*change_me[[:space:]]*$' "$ENV_FILE" && [ -z "${DB_PASSWORD:-}" ]; then
  echo "DB_PASSWORD in backend/.env is still change_me. Please set the real password." >&2
  exit 1
fi

if [ "$SKIP_INSTALL" != "1" ] && [ ! -d "$FRONTEND/node_modules" ]; then
  echo "frontend/node_modules not found. Installing frontend dependencies..."
  (cd "$FRONTEND" && npm install)
fi

if [ "$SKIP_MIGRATE" != "1" ]; then
  echo "Running database migrations..."
  (cd "$BACKEND" && go run ./cmd/migrate)
fi

echo
echo "Temu Tools is starting..."
echo "Backend API: http://localhost:8080"
echo "Frontend: http://localhost:5173"
echo "Press Ctrl+C to stop both services."
echo

(cd "$BACKEND" && go run ./cmd/server) &
BACKEND_PID=$!

(cd "$FRONTEND" && npm run dev -- --host 0.0.0.0) &
FRONTEND_PID=$!

while true; do
  if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    echo "Backend service exited. Please check the logs above." >&2
    exit 1
  fi
  if ! kill -0 "$FRONTEND_PID" 2>/dev/null; then
    echo "Frontend service exited. Please check the logs above." >&2
    exit 1
  fi
  sleep 1
done
