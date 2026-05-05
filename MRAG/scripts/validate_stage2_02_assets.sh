#!/usr/bin/env bash
set -euo pipefail

# shellcheck disable=SC1091
source "$(cd "$(dirname "$0")" && pwd)/validate_stage2_common.sh"
load_state

log "creating stage2 validation research assets"
IDEA_RESP="$(cat <<EOF | api_post "/ideas"
{"title":"Stage2 Validation Idea ${STAMP}","descriptionMd":"Validation idea for stage2 automation.","status":"draft","weight":70,"priority":80,"confidence":0.7,"sourceType":"human","sourceNote":"stage2 validation"}
EOF
)"
IDEA_ID="$(printf '%s' "${IDEA_RESP}" | json_query data.idea.id)"

ASSET_RESP="$(cat <<EOF | api_post "/dataset-assets"
{"name":"Stage2 Validation Dataset ${STAMP}","descriptionMd":"Synthetic validation dataset asset.","taskType":"text","status":"active","sourceType":"manual","localOrRemotePath":"/tmp/stage2-validation-${STAMP}","readmeMd":"validation readme","loaderNoteMd":"validation loader","schemaNoteMd":"validation schema"}
EOF
)"
DATASET_ASSET_ID="$(printf '%s' "${ASSET_RESP}" | json_query data.asset.id)"

BASELINE_RESP="$(cat <<EOF | api_post "/baselines"
{"datasetAssetId":"${DATASET_ASSET_ID}","name":"Stage2 Validation Baseline","metricSchemaJson":{"primary":"accuracy","secondary":["loss"]},"resultJson":{"accuracy":0.72,"loss":0.34},"noteMd":"validation baseline","sourceType":"manual"}
EOF
)"
BASELINE_ID="$(printf '%s' "${BASELINE_RESP}" | json_query data.baseline.id)"

ARCHIVE_RESP="$(cat <<EOF | api_post "/result-archives"
{"title":"Stage2 Validation Archive","datasetAssetId":"${DATASET_ASSET_ID}","baselineId":"${BASELINE_ID}","ideaId":"${IDEA_ID}","summaryMd":"historic validation result","metricJson":{"accuracy":0.79,"loss":0.27},"status":"archived","noteMd":"historic run","files":[{"fileName":"figure.txt","fileKind":"figure","content":"stage2 validation figure"}]}
EOF
)"
ARCHIVE_ID="$(printf '%s' "${ARCHIVE_RESP}" | json_query data.archive.id)"
[[ -n "${ARCHIVE_ID}" ]]

log "creating mock server and validating heartbeat/gpu snapshots"
SERVER_RESP="$(cat <<EOF | api_post "/servers"
{"name":"Stage2 Mock Server ${STAMP}","host":"mock-online-${STAMP}","sshPort":22,"username":"demo","authType":"ssh_config","remoteRoot":"/tmp/mrag","taskWorkdir":"/tmp/mrag/tasks","config":{"profile":"stage2-validation"}}
EOF
)"
SERVER_ID="$(printf '%s' "${SERVER_RESP}" | json_query data.id)"

HB_RESP="$(printf '{}' | api_post "/servers/${SERVER_ID}/heartbeat")"
HB_STATUS="$(printf '%s' "${HB_RESP}" | json_query data.heartbeat.status)"
[[ "${HB_STATUS}" == "online" ]]

HB_LIST="$(curl.exe -fsS "${API_BASE}/servers/${SERVER_ID}/heartbeats")"
HB_COUNT="$(printf '%s' "${HB_LIST}" | json_query data | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[[ "${HB_COUNT}" -ge 1 ]]

GPU_RESP="$(printf '{}' | api_post "/servers/${SERVER_ID}/gpu-snapshot")"
GPU_TOTAL="$(printf '%s' "${GPU_RESP}" | json_query data.totalGpuCount)"
[[ "${GPU_TOTAL}" -ge 1 ]]

GPU_LIST="$(curl.exe -fsS "${API_BASE}/servers/${SERVER_ID}/gpu-snapshots")"
GPU_COUNT="$(printf '%s' "${GPU_LIST}" | json_query data | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[[ "${GPU_COUNT}" -ge 1 ]]

write_state
cat <<EOF
PASS: stage2 asset and resource validation passed
- idea_id: ${IDEA_ID}
- dataset_asset_id: ${DATASET_ASSET_ID}
- baseline_id: ${BASELINE_ID}
- archive_id: ${ARCHIVE_ID}
- server_id: ${SERVER_ID}
EOF
