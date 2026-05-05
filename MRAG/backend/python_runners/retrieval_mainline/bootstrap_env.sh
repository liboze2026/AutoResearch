#!/usr/bin/env bash
set -euo pipefail

python -m pip install --upgrade pip
python -m pip install -r requirements.txt 2>/dev/null || true
echo "phase4 retrieval mainline bootstrap complete"
