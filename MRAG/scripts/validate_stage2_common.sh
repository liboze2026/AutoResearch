#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
API_BASE="${API_BASE:-http://127.0.0.1:18080/api/v1}"
FRONTEND_BASE="${FRONTEND_BASE:-http://127.0.0.1:4173}"
PSH_BIN="${PSH_BIN:-powershell.exe}"
HTTP_BIN="${HTTP_BIN:-curl.exe}"
STATE_FILE="${ROOT_DIR}/.stage2_validate_state"
ROOT_DIR_WIN="$(wslpath -w "${ROOT_DIR}")"
STAMP="${STAMP:-$(date +%s)}"
BACKEND_CONTAINER="${BACKEND_CONTAINER:-mrag-stage2-backend-validation}"
FRONTEND_LOG="${ROOT_DIR}/.stage2-frontend.log"
FRONTEND_ERR_LOG="${ROOT_DIR}/.stage2-frontend.err.log"
FRONTEND_LOG_WIN="$(wslpath -w "${FRONTEND_LOG}")"
FRONTEND_ERR_LOG_WIN="$(wslpath -w "${FRONTEND_ERR_LOG}")"

log() {
  printf '[stage2] %s\n' "$1"
}

ps_run() {
  "${PSH_BIN}" -Command "$1"
}

wait_http() {
  local url="$1"
  local retries="${2:-90}"
  local sleep_sec="${3:-2}"
  for _ in $(seq 1 "$retries"); do
    if "${HTTP_BIN}" -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$sleep_sec"
  done
  return 1
}

json_query() {
  local path="$1"
  python3 -c 'import json, sys; path = sys.argv[1].split("."); data = json.load(sys.stdin); 
for token in path:
    if token == "":
        continue
    if token.isdigit():
        data = data[int(token)]
    else:
        data = data[token]
print(json.dumps(data, ensure_ascii=False) if isinstance(data, (dict, list)) else data)' "$path"
}

api_post() {
  local path="$1"
  python3 -c 'import sys, urllib.request; url = sys.argv[1]; body = sys.stdin.buffer.read(); req = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"}, method="POST"); resp = urllib.request.urlopen(req); sys.stdout.write(resp.read().decode("utf-8"))' "${API_BASE}${path}"
}

write_state() {
  cat > "${STATE_FILE}" <<EOF
export ROOT_DIR='${ROOT_DIR}'
export API_BASE='${API_BASE}'
export FRONTEND_BASE='${FRONTEND_BASE}'
export STAMP='${STAMP}'
export BACKEND_CONTAINER='${BACKEND_CONTAINER}'
export FRONTEND_PID='${FRONTEND_PID:-}'
export IDEA_ID='${IDEA_ID:-}'
export DATASET_ASSET_ID='${DATASET_ASSET_ID:-}'
export BASELINE_ID='${BASELINE_ID:-}'
export ARCHIVE_ID='${ARCHIVE_ID:-}'
export SERVER_ID='${SERVER_ID:-}'
export EXPERIMENT_ID='${EXPERIMENT_ID:-}'
export RUN_ID='${RUN_ID:-}'
export RETRY_RUN_ID='${RETRY_RUN_ID:-}'
export RUN_RESULT_ARCHIVE_ID='${RUN_RESULT_ARCHIVE_ID:-}'
EOF
}

load_state() {
  if [[ -f "${STATE_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${STATE_FILE}"
  fi
}
