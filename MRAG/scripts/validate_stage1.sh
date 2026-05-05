#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
API_BASE="${API_BASE:-http://127.0.0.1:8080/api/v1}"
FRONTEND_BASE="${FRONTEND_BASE:-http://127.0.0.1:5173}"
FRONTEND_PID=""

log() {
  printf '[stage1] %s\n' "$1"
}

wait_http() {
  local url="$1"
  local retries="${2:-60}"
  local sleep_sec="${3:-2}"
  for _ in $(seq 1 "$retries"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$sleep_sec"
  done
  return 1
}

json_query() {
  local path="$1"
  python3 - "$path" <<'PY'
import json, sys
path = sys.argv[1].split('.')
data = json.load(sys.stdin)
for token in path:
    if token.isdigit():
        data = data[int(token)]
    else:
        data = data[token]
if isinstance(data, (dict, list)):
    print(json.dumps(data, ensure_ascii=False))
else:
    print(data)
PY
}

cleanup() {
  if [[ -n "${FRONTEND_PID}" ]]; then
    kill "${FRONTEND_PID}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

cd "$ROOT_DIR"

log "starting postgres and backend"
docker compose up -d postgres >/dev/null
if ! docker compose up -d go-backend >/dev/null 2>&1; then
  log "existing go-backend image missing or stale, trying compose build"
  docker compose up -d --build go-backend >/dev/null
fi

wait_http "http://127.0.0.1:8080/healthz" 90 2
log "backend is reachable"

log "running backend service tests"
(
  cd backend/go
  go test ./internal/service
) >/dev/null
log "backend service tests passed"

log "running frontend basic tests"
npm run test:frontend:basic >/dev/null
log "frontend typecheck/build passed"

if ! curl -fsS "$FRONTEND_BASE" >/dev/null 2>&1; then
  log "starting frontend dev server"
  npm run dev -- --host 127.0.0.1 > "$ROOT_DIR/.stage1-frontend.log" 2>&1 &
  FRONTEND_PID=$!
  wait_http "$FRONTEND_BASE" 90 2
else
  log "frontend already running"
fi
log "frontend is reachable"

STAMP="$(date +%s)"
PAPER_FILE_HOST="$ROOT_DIR/workspace/papers/incoming/stage1_validation_${STAMP}.md"
DATASET_DIR_HOST="$ROOT_DIR/workspace/datasets/stage1_validation_${STAMP}"
mkdir -p "$(dirname "$PAPER_FILE_HOST")" "$DATASET_DIR_HOST/train"
cat > "$PAPER_FILE_HOST" <<EOF
# Stage1 Validation Paper ${STAMP}
This is a deterministic validation paper for stage1 acceptance.
EOF
cat > "$DATASET_DIR_HOST/train/sample.jsonl" <<EOF
{"prompt": "hello", "answer": "world"}
EOF

PAPER_FILE_CONTAINER="/app/workspace/papers/incoming/$(basename "$PAPER_FILE_HOST")"
DATASET_DIR_CONTAINER="/app/workspace/datasets/$(basename "$DATASET_DIR_HOST")"

log "importing paper"
IMPORT_RESP="$(curl -fsS -X POST "$API_BASE/papers/import" -H 'Content-Type: application/json' -d "{\"existingPath\":\"$PAPER_FILE_CONTAINER\"}")"
PAPER_ID="$(printf '%s' "$IMPORT_RESP" | json_query data.paper.id)"
[[ -n "$PAPER_ID" ]]
log "paper import ok: $PAPER_ID"

PARSE_RESP="$(curl -fsS -X POST "$API_BASE/papers/$PAPER_ID/parse" -H 'Content-Type: application/json' -d '{}')"
PARSE_STATUS="$(printf '%s' "$PARSE_RESP" | json_query data.paper.status)"
[[ "$PARSE_STATUS" == "parsed" || "$PARSE_STATUS" == "insight_extracted" ]]
log "paper parse ok"

EXTRACT_RESP="$(curl -fsS -X POST "$API_BASE/papers/$PAPER_ID/extract-insights" -H 'Content-Type: application/json' -d '{}')"
EXTRACT_STATUS="$(printf '%s' "$EXTRACT_RESP" | json_query data.insight.extractStatus)"
[[ "$EXTRACT_STATUS" == "completed" ]]
log "paper insight extraction ok"

IDEA_CREATE_RESP="$(curl -fsS -X POST "$API_BASE/ideas" -H 'Content-Type: application/json' -d '{"title":"Stage1 Validation Idea","descriptionMd":"Validation idea.","status":"draft","weight":80,"priority":85,"confidence":0.8,"sourceType":"human","sourceNote":"validation note"}')"
IDEA_ID="$(printf '%s' "$IDEA_CREATE_RESP" | json_query data.idea.id)"
[[ -n "$IDEA_ID" ]]
log "manual idea create ok: $IDEA_ID"

IDEA_GEN_RESP="$(curl -fsS -X POST "$API_BASE/ideas/generate-from-paper/$PAPER_ID" -H 'Content-Type: application/json' -d '{}')"
IDEA_GEN_COUNT="$(printf '%s' "$IDEA_GEN_RESP" | json_query data.ideas | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
[[ "$IDEA_GEN_COUNT" -ge 2 ]]
log "idea generate ok"

log "creating MRAG scanned dataset via existing dataset import flow"
DATASET_RESP="$(curl -fsS -X POST "$API_BASE/datasets" -H 'Content-Type: application/json' -d "{\"name\":\"Stage1 Validation Dataset $STAMP\",\"sourceType\":\"local\",\"path\":\"$DATASET_DIR_CONTAINER\",\"description\":\"stage1 validation dataset\",\"tags\":[\"stage1\",\"validation\"],\"modality\":\"text\"}")"
DATASET_ID="$(printf '%s' "$DATASET_RESP" | json_query data.dataset.id)"
SCAN_RECORD_ID="$(printf '%s' "$DATASET_RESP" | json_query data.latestScan.id)"
[[ -n "$DATASET_ID" ]]
[[ -n "$SCAN_RECORD_ID" ]]
log "dataset import/scan ok: $DATASET_ID"

ASSET_RESP="$(curl -fsS -X POST "$API_BASE/dataset-assets/register-from-scan" -H 'Content-Type: application/json' -d "{\"scanRecordId\":\"$SCAN_RECORD_ID\"}")"
DATASET_ASSET_ID="$(printf '%s' "$ASSET_RESP" | json_query data.asset.id)"
[[ -n "$DATASET_ASSET_ID" ]]
log "dataset asset register ok: $DATASET_ASSET_ID"

BASELINE_RESP="$(curl -fsS -X POST "$API_BASE/baselines" -H 'Content-Type: application/json' -d "{\"datasetAssetId\":\"$DATASET_ASSET_ID\",\"name\":\"Validation Baseline\",\"metricSchemaJson\":{\"primary\":\"accuracy\"},\"resultJson\":{\"accuracy\":0.77},\"noteMd\":\"validation baseline\",\"sourceType\":\"manual\"}")"
BASELINE_ID="$(printf '%s' "$BASELINE_RESP" | json_query data.baseline.id)"
[[ -n "$BASELINE_ID" ]]
log "baseline create ok: $BASELINE_ID"

ARCHIVE_RESP="$(curl -fsS -X POST "$API_BASE/result-archives" -H 'Content-Type: application/json' -d "{\"title\":\"Validation Archive\",\"datasetAssetId\":\"$DATASET_ASSET_ID\",\"ideaId\":\"$IDEA_ID\",\"summaryMd\":\"validation archive summary\",\"metricJson\":{\"accuracy\":0.81},\"status\":\"archived\",\"noteMd\":\"validation archive\",\"files\":[{\"fileName\":\"figure.txt\",\"fileKind\":\"figure\",\"content\":\"figure placeholder\"}]}")"
ARCHIVE_ID="$(printf '%s' "$ARCHIVE_RESP" | json_query data.archive.id)"
[[ -n "$ARCHIVE_ID" ]]
log "result archive create ok: $ARCHIVE_ID"

log "checking key pages"
for path in / /papers "/papers/$PAPER_ID" /ideas /dataset-assets /baselines /result-archives; do
  code="$(curl -o /dev/null -s -w '%{http_code}' "$FRONTEND_BASE$path")"
  [[ "$code" == "200" ]]
done
log "frontend pages reachable"

cat <<EOF

Stage1 validation passed.
- paper_id: $PAPER_ID
- idea_id: $IDEA_ID
- dataset_id: $DATASET_ID
- dataset_asset_id: $DATASET_ASSET_ID
- baseline_id: $BASELINE_ID
- archive_id: $ARCHIVE_ID
EOF