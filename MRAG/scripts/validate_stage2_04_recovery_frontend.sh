#!/usr/bin/env bash
set -euo pipefail

# shellcheck disable=SC1091
source "$(cd "$(dirname "$0")" && pwd)/validate_stage2_common.sh"
load_state

[[ -n "${EXPERIMENT_ID:-}" ]]
[[ -n "${RUN_ID:-}" ]]
[[ -n "${SERVER_ID:-}" ]]

log "simulating failed run and validating retry"
FAILED_RUN_ID="run_stage2_failed_$(date +%s)"
docker compose exec -T postgres psql -U postgres -d mrag_platform -c "INSERT INTO experiment_runs (id, experiment_id, spec_id, assigned_server_id, run_status, remote_workdir, retry_count, exit_code, result_json, error_message, created_at, updated_at) VALUES ('${FAILED_RUN_ID}', '${EXPERIMENT_ID}', (SELECT id FROM experiment_specs WHERE experiment_id='${EXPERIMENT_ID}' ORDER BY version DESC LIMIT 1), '${SERVER_ID}', 'failed', '/tmp/mrag/experiments/${EXPERIMENT_ID}/run_failed', 0, 1, '{\"failure_stage\":\"runner_exit\",\"last_log_summary\":\"mock failure tail\",\"recovery\":{\"suggest_retry\":true}}', 'mock failure for validation', now(), now());" >/dev/null

RECOVERY_RESP="$(curl.exe -fsS "${API_BASE}/runs/${FAILED_RUN_ID}/recovery")"
RECOVERY_STAGE="$(printf '%s' "${RECOVERY_RESP}" | json_query data.failureStage)"
RECOVERY_RETRY="$(printf '%s' "${RECOVERY_RESP}" | json_query data.suggestRetry)"
[[ "${RECOVERY_STAGE}" == "runner_exit" ]]
[[ "${RECOVERY_RETRY}" == "True" || "${RECOVERY_RETRY}" == "true" ]]

RETRY_RESP="$(printf '{}' | api_post "/runs/${FAILED_RUN_ID}/retry")"
RETRY_RUN_ID="$(printf '%s' "${RETRY_RESP}" | json_query data.run.id)"
RETRY_STATUS="$(printf '%s' "${RETRY_RESP}" | json_query data.run.runStatus)"
RETRY_COUNT="$(printf '%s' "${RETRY_RESP}" | json_query data.run.retryCount)"
[[ -n "${RETRY_RUN_ID}" ]]
[[ "${RETRY_STATUS}" == "queued" ]]
[[ "${RETRY_COUNT}" -ge 1 ]]

log "checking frontend routes"
for path in / /experiments "/experiments/${EXPERIMENT_ID}" "/experiments/${EXPERIMENT_ID}/comparisons" /servers; do
  code="$(curl.exe -o NUL -s -w '%{http_code}' "${FRONTEND_BASE}${path}")"
  [[ "${code}" == "200" ]]
done

write_state
cat <<EOF
PASS: stage2 recovery and frontend validation passed
- failed_run_id: ${FAILED_RUN_ID}
- retry_run_id: ${RETRY_RUN_ID}
EOF
