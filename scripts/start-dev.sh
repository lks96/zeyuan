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
BACKGROUND="${BACKGROUND:-0}"
COMMAND="start"
PID_DIR="${PID_DIR:-$ROOT/.run}"
LOG_DIR="${LOG_DIR:-$ROOT/logs}"
BACKEND_PID_FILE="$PID_DIR/backend.pid"
FRONTEND_PID_FILE="$PID_DIR/frontend.pid"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"
BACKEND_PID=""
FRONTEND_PID=""

usage() {
  cat <<EOF
Usage: bash ./scripts/start-dev.sh [start] [--background]
       bash ./scripts/start-dev.sh stop
       bash ./scripts/start-dev.sh status

Options:
  -d, --background    Start backend and frontend in the background.
  -h, --help          Show this help.

Environment:
  SKIP_INSTALL=1      Skip frontend dependency install.
  SKIP_MIGRATE=1      Skip database migrations.
  PID_DIR=path        Override pid directory. Default: $PID_DIR
  LOG_DIR=path        Override log directory. Default: $LOG_DIR
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    start)
      COMMAND="start"
      ;;
    stop|status)
      COMMAND="$1"
      ;;
    -d|--background)
      BACKGROUND="1"
      ;;
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

read_pid() {
  local pid_file="$1"
  if [ -f "$pid_file" ]; then
    cat "$pid_file"
  fi
}

is_running() {
  local pid_file="$1"
  local pid
  pid="$(read_pid "$pid_file")"
  [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null
}

print_service_status() {
  local name="$1"
  local pid_file="$2"
  local pid
  pid="$(read_pid "$pid_file")"

  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    echo "$name: running (pid $pid)"
  elif [ -n "$pid" ]; then
    echo "$name: stopped (stale pid $pid)"
  else
    echo "$name: stopped"
  fi
}

status_services() {
  print_service_status "Backend" "$BACKEND_PID_FILE"
  print_service_status "Frontend" "$FRONTEND_PID_FILE"
}

stop_service() {
  local name="$1"
  local pid_file="$2"
  local pid
  pid="$(read_pid "$pid_file")"

  if [ -z "$pid" ]; then
    echo "$name is not running."
    return
  fi

  if kill -0 "$pid" 2>/dev/null; then
    echo "Stopping $name (pid $pid)..."
    kill "$pid" 2>/dev/null || true
  else
    echo "$name is not running."
  fi

  rm -f "$pid_file"
}

stop_services() {
  stop_service "backend" "$BACKEND_PID_FILE"
  stop_service "frontend" "$FRONTEND_PID_FILE"
}

cleanup() {
  if [ -n "$BACKEND_PID" ] && kill -0 "$BACKEND_PID" 2>/dev/null; then
    kill "$BACKEND_PID" 2>/dev/null || true
  fi
  if [ -n "$FRONTEND_PID" ] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
    kill "$FRONTEND_PID" 2>/dev/null || true
  fi
}

if [ "$COMMAND" = "status" ]; then
  status_services
  exit 0
fi

if [ "$COMMAND" = "stop" ]; then
  stop_services
  exit 0
fi

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
  echo "Preparing database schema..."
  (cd "$BACKEND" && go run ./cmd/dbprepare)
fi

if is_running "$BACKEND_PID_FILE" || is_running "$FRONTEND_PID_FILE"; then
  echo "Temu Tools already appears to be running:" >&2
  status_services >&2
  echo "Run 'bash ./scripts/start-dev.sh stop' before starting it again." >&2
  exit 1
fi

rm -f "$BACKEND_PID_FILE" "$FRONTEND_PID_FILE"

echo
echo "Temu Tools is starting..."
echo "Backend API: http://localhost:8080"
echo "Frontend: http://localhost:5173"

if [ "$BACKGROUND" = "1" ]; then
  mkdir -p "$PID_DIR" "$LOG_DIR"

  echo "Starting in background..."
  echo "Backend log: $BACKEND_LOG"
  echo "Frontend log: $FRONTEND_LOG"
  echo

  (cd "$BACKEND" && nohup go run ./cmd/server >>"$BACKEND_LOG" 2>&1 </dev/null & echo $! >"$BACKEND_PID_FILE")
  (cd "$FRONTEND" && nohup npm run dev -- --host 0.0.0.0 >>"$FRONTEND_LOG" 2>&1 </dev/null & echo $! >"$FRONTEND_PID_FILE")

  sleep 2

  if ! is_running "$BACKEND_PID_FILE" || ! is_running "$FRONTEND_PID_FILE"; then
    echo "One or more services failed to stay running. Check the logs above." >&2
    status_services >&2
    stop_services >&2
    exit 1
  fi

  status_services
  echo
  echo "Stop services with: bash ./scripts/start-dev.sh stop"
  exit 0
fi

echo "Press Ctrl+C to stop both services."
echo

trap cleanup EXIT INT TERM

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
