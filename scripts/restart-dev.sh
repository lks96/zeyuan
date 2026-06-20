#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
START_SCRIPT="$SCRIPT_DIR/start-dev.sh"
PORTS=(8080 5173)

SKIP_INSTALL="${SKIP_INSTALL:-0}"
SKIP_MIGRATE="${SKIP_MIGRATE:-0}"

usage() {
  cat <<EOF
Usage: bash ./scripts/restart-dev.sh

Environment:
  SKIP_INSTALL=1      Skip frontend dependency install.
  SKIP_MIGRATE=1      Skip database migrations.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

kill_port_listeners() {
  local port="$1"
  local pids=""

  if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  elif command -v fuser >/dev/null 2>&1; then
    pids="$(fuser "$port/tcp" 2>/dev/null || true)"
  fi

  if [ -z "$pids" ]; then
    return
  fi

  echo "Stopping listener(s) on port $port: $pids"
  kill $pids 2>/dev/null || true
}

wait_for_ports_to_close() {
  local deadline=$((SECONDS + 10))
  local open

  while [ "$SECONDS" -lt "$deadline" ]; do
    open=0
    for port in "${PORTS[@]}"; do
      if command -v lsof >/dev/null 2>&1 && lsof -tiTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
        open=1
      elif command -v fuser >/dev/null 2>&1 && fuser "$port/tcp" >/dev/null 2>&1; then
        open=1
      fi
    done

    if [ "$open" = "0" ]; then
      return
    fi

    sleep 1
  done
}

echo "Restarting Temu Tools dev services..."

bash "$START_SCRIPT" stop || true

for port in "${PORTS[@]}"; do
  kill_port_listeners "$port"
done

wait_for_ports_to_close

SKIP_INSTALL="$SKIP_INSTALL" SKIP_MIGRATE="$SKIP_MIGRATE" bash "$START_SCRIPT" start --background

echo
echo "Temu Tools restarted."
echo "Backend API: http://localhost:8080"
echo "Frontend: http://localhost:5173"
