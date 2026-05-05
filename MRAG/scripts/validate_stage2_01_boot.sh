#!/usr/bin/env bash
set -euo pipefail

# shellcheck disable=SC1091
source "$(cd "$(dirname "$0")" && pwd)/validate_stage2_common.sh"

log "starting postgres"
docker compose up -d postgres >/dev/null

log "running backend stage2 package tests"
ps_run "Set-Location '${ROOT_DIR_WIN}\\backend\\go'; \$env:GOCACHE='${ROOT_DIR_WIN}\\.gocache'; \$env:GOMODCACHE='${ROOT_DIR_WIN}\\.gomodcache'; go test -buildvcs=false ./internal/heartbeat ./internal/gpuresource ./internal/recovery ./internal/resultcompare ./internal/runner ./internal/handler ./internal/scheduler | Out-Null"
log "backend stage2 tests passed"

log "running frontend basic tests"
ps_run "Set-Location '${ROOT_DIR_WIN}'; npm run test:frontend:basic | Out-Null"
log "frontend basic tests passed"

log "building backend image if needed"
docker compose build go-backend >/dev/null

log "restarting mock backend container"
docker ps -aq --filter "name=mrag-stage2-backend" | xargs -r docker rm -f >/dev/null 2>&1 || true
docker rm -f "${BACKEND_CONTAINER}" >/dev/null 2>&1 || true
docker compose run -d --name "${BACKEND_CONTAINER}" \
  -p 18080:8080 \
  -e APP_PORT=8080 \
  -e POSTGRES_DSN='postgres://postgres:root@postgres:5432/mrag_platform?sslmode=disable' \
  -e WORKSPACE_ROOT=/app/workspace \
  -e SSH_CLIENT_MODE=mock \
  -e SSH_DIAL_TIMEOUT_SEC=2 \
  -e SSH_COMMAND_TIMEOUT_SEC=10 \
  -e REMOTE_EXECUTION_MODE=mock \
  -e REMOTE_WORK_ROOT=/tmp/mrag \
  -e DATASET_SCAN_MODE=mock \
  -e DATASET_INDEX_MODE=mock \
  -e OVERVIEW_STATS_MODE=mock \
  -e SERVER_HEARTBEAT_INTERVAL_SEC=0 \
  -e GPU_SNAPSHOT_INTERVAL_SEC=0 \
  go-backend >/dev/null

wait_http "http://127.0.0.1:18080/healthz" 120 2
log "backend is reachable"

log "starting frontend dev server"
ps_run "Get-Process | Where-Object { \$_.ProcessName -like 'node*' -and \$_.Path -like '*node.exe*' } | Stop-Process -Force -ErrorAction SilentlyContinue"
FRONTEND_PID="$(ps_run "\$cmd = 'set VITE_API_BASE_URL=${API_BASE} && cd /d ${ROOT_DIR_WIN} && npm.cmd run dev -- --host 127.0.0.1 --port 4173 > ${FRONTEND_LOG_WIN} 2> ${FRONTEND_ERR_LOG_WIN}'; \$p = Start-Process -FilePath 'cmd.exe' -ArgumentList '/c', \$cmd -WindowStyle Hidden -PassThru; [Console]::Out.WriteLine(\$p.Id)")"
wait_http "${FRONTEND_BASE}" 120 2
log "frontend is reachable"

write_state
cat <<EOF
PASS: stage2 boot validation passed
- backend_container: ${BACKEND_CONTAINER}
- frontend_pid: ${FRONTEND_PID}
EOF
