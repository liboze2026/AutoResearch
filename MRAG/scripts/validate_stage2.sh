#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

run_step() {
  local script_name="$1"
  printf '[stage2] running %s\n' "${script_name}"
  bash "${SCRIPT_DIR}/${script_name}"
}

run_step "validate_stage2_01_boot.sh"
run_step "validate_stage2_02_assets.sh"
run_step "validate_stage2_03_lifecycle.sh"
run_step "validate_stage2_04_recovery_frontend.sh"

cat <<'EOF'
PASS: stage2 validation passed
- steps:
  - validate_stage2_01_boot.sh
  - validate_stage2_02_assets.sh
  - validate_stage2_03_lifecycle.sh
  - validate_stage2_04_recovery_frontend.sh
EOF
