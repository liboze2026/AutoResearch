#!/usr/bin/env bash
set -euo pipefail

# shellcheck disable=SC1091
source "$(cd "$(dirname "$0")" && pwd)/validate_stage2_common.sh"
load_state

[[ -n "${DATASET_ASSET_ID:-}" ]]
[[ -n "${BASELINE_ID:-}" ]]
[[ -n "${IDEA_ID:-}" ]]
[[ -n "${SERVER_ID:-}" ]]

log "creating experiment and generating spec"
EXPERIMENT_RESP="$(cat <<EOF | api_post "/experiments"
{"datasetAssetId":"${DATASET_ASSET_ID}","ideaId":"${IDEA_ID}","baselineId":"${BASELINE_ID}","title":"Stage2 Validation Experiment ${STAMP}","priority":88,"summaryMd":"stage2 validation experiment"}
EOF
)"
EXPERIMENT_ID="$(printf '%s' "${EXPERIMENT_RESP}" | json_query data.experiment.id)"

SPEC_RESP="$(printf '{}' | api_post "/experiments/${EXPERIMENT_ID}/generate-spec")"
TEMPLATE_TYPE="$(printf '%s' "${SPEC_RESP}" | json_query data.spec.templateType)"
[[ -n "${TEMPLATE_TYPE}" ]]

log "queueing and scheduling run"
QUEUE_RESP="$(printf '{}' | api_post "/experiments/${EXPERIMENT_ID}/queue")"
RUN_ID="$(printf '%s' "${QUEUE_RESP}" | json_query data.run.id)"
RUN_STATUS="$(printf '%s' "${QUEUE_RESP}" | json_query data.run.runStatus)"
[[ "${RUN_STATUS}" == "queued" ]]

SCHEDULE_RESP="$(printf '{}' | api_post "/runs/${RUN_ID}/schedule")"
SCHEDULED_SERVER_ID="$(printf '%s' "${SCHEDULE_RESP}" | json_query data.run.assignedServerId)"
SCHEDULED_STATUS="$(printf '%s' "${SCHEDULE_RESP}" | json_query data.run.runStatus)"
[[ "${SCHEDULED_STATUS}" == "scheduled" ]]
[[ "${SCHEDULED_SERVER_ID}" == "${SERVER_ID}" ]]

log "starting run and validating logs"
START_RESP="$(printf '{}' | api_post "/runs/${RUN_ID}/start")"
START_STATUS="$(printf '%s' "${START_RESP}" | json_query data.runStatus)"
[[ "${START_STATUS}" == "succeeded" ]]

RUN_GET="$(curl.exe -fsS "${API_BASE}/runs/${RUN_ID}")"
RUN_RESULT_ARCHIVE_ID="$(printf '%s' "${RUN_GET}" | json_query data.resultJson.result_archive_id)"
[[ -n "${RUN_RESULT_ARCHIVE_ID}" ]]

RUN_LOGS="$(curl.exe -fsS "${API_BASE}/runs/${RUN_ID}/logs")"
RUN_LOG_COUNT="$(printf '%s' "${RUN_LOGS}" | json_query data | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[[ "${RUN_LOG_COUNT}" -ge 2 ]]

RUN_TAIL="$(curl.exe -fsS "${API_BASE}/runs/${RUN_ID}/logs/tail?type=stdout")"
RUN_TAIL_TEXT="$(printf '%s' "${RUN_TAIL}" | json_query data.tail)"
[[ "${RUN_TAIL_TEXT}" == *"mock-train"* ]]

log "validating comparison generation"
COMPARE_RESP="$(printf '{}' | api_post "/runs/${RUN_ID}/compare")"
COMPARE_COUNT="$(printf '%s' "${COMPARE_RESP}" | json_query data.comparisons | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[[ "${COMPARE_COUNT}" -ge 1 ]]

EXPERIMENT_COMPARE_LIST="$(curl.exe -fsS "${API_BASE}/experiments/${EXPERIMENT_ID}/comparisons")"
EXPERIMENT_COMPARE_COUNT="$(printf '%s' "${EXPERIMENT_COMPARE_LIST}" | json_query data | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[[ "${EXPERIMENT_COMPARE_COUNT}" -ge 1 ]]

write_state
cat <<EOF
PASS: stage2 lifecycle validation passed
- experiment_id: ${EXPERIMENT_ID}
- run_id: ${RUN_ID}
- result_archive_id: ${RUN_RESULT_ARCHIVE_ID}
- comparison_count: ${EXPERIMENT_COMPARE_COUNT}
EOF
