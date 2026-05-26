#!/usr/bin/env bash
set -euo pipefail

repo_root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  cd "$script_dir/.." && pwd
}

ROOT="${HERMES_VOICE_ROOT:-$(repo_root)}"
STATE_DIR="${HERMES_VOICE_STATE_DIR:-$ROOT/.hermes/dev-voice}"
LOG_DIR="$STATE_DIR/logs"
PID_DIR="$STATE_DIR/pids"
BIN_DIR="$STATE_DIR/bin"
mkdir -p "$LOG_DIR" "$PID_DIR" "$BIN_DIR"

GO_BIN="${GO_BIN:-/home/sve/.local/go/bin/go}"
HERMES_BIN="${HERMES_BIN:-/home/sve/.local/bin/hermes}"
CENTRAL_LISTEN="${CENTRAL_LISTEN:-0.0.0.0:18081}"
CENTRAL_LOCAL_URL="${CENTRAL_LOCAL_URL:-http://127.0.0.1:18081}"
CENTRAL_LAN_URL="${CENTRAL_LAN_URL:-http://192.168.7.50:18081}"
CENTRAL_SOURCE="${CENTRAL_SOURCE:-hermes-voice-central-dev}"
CENTRAL_QUICK_TIMEOUT="${CENTRAL_QUICK_TIMEOUT:-15s}"
CENTRAL_HERMES_TIMEOUT="${CENTRAL_HERMES_TIMEOUT:-180s}"
CENTRAL_MAX_TURNS="${CENTRAL_MAX_TURNS:-3}"

EDGE_HOST="${EDGE_HOST:-sve@192.168.7.72}"
EDGE_SSH_KEY="${EDGE_SSH_KEY:-/home/sve/.hermes-external-memory/software-dev-projects/hermes-voice/.private/ssh/orange-pi-3b/id_ed25519_hermes_orangepi}"
EDGE_BINARY="${EDGE_BINARY:-/home/sve/hermes-voice-forwarder}"
EDGE_LISTEN="${EDGE_LISTEN:-127.0.0.1:8081}"
EDGE_LOCAL_URL="${EDGE_LOCAL_URL:-http://127.0.0.1:8081}"
EDGE_ID="${EDGE_ID:-orange-pi-ha}"
EDGE_ROOM="${EDGE_ROOM:-cabinet}"
EDGE_DEVICE_ID="${EDGE_DEVICE_ID:-phone_ha}"
EDGE_TIMEOUT="${EDGE_TIMEOUT:-30s}"
EDGE_UPSTREAM="${EDGE_UPSTREAM:-$CENTRAL_LAN_URL}"

CENTRAL_PID_FILE="$PID_DIR/central.pid"
EDGE_PID_FILE="$PID_DIR/edge.pid"
CENTRAL_LOG="$LOG_DIR/central.log"
EDGE_LOG="$LOG_DIR/edge.log"
EDGE_BINARY_LOCAL="$BIN_DIR/hermes-voice-linux-arm64"

ssh_edge() {
  ssh -i "$EDGE_SSH_KEY" -o IdentitiesOnly=yes -o BatchMode=yes "$EDGE_HOST" "$@"
}

scp_edge() {
  scp -i "$EDGE_SSH_KEY" -o IdentitiesOnly=yes -o BatchMode=yes "$@"
}

is_pid_alive() {
  local pid="$1"
  [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

wait_http_ok() {
  local url="$1"
  local attempts="${2:-30}"
  local i
  for i in $(seq 1 "$attempts"); do
    if curl -fsS --connect-timeout 2 "$url" >/dev/null; then
      return 0
    fi
    sleep 1
  done
  return 1
}

print_config() {
  cat <<EOF
ROOT=$ROOT
STATE_DIR=$STATE_DIR
CENTRAL_LISTEN=$CENTRAL_LISTEN
CENTRAL_LOCAL_URL=$CENTRAL_LOCAL_URL
CENTRAL_LAN_URL=$CENTRAL_LAN_URL
EDGE_HOST=$EDGE_HOST
EDGE_LISTEN=$EDGE_LISTEN
EDGE_UPSTREAM=$EDGE_UPSTREAM
EDGE_BINARY=$EDGE_BINARY
EOF
}
