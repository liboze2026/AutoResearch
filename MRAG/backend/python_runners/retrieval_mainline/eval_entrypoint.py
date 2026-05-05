#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json

from mainline import load_predictions
from protocol import load_config, load_manifest
from tools.evaluate_tool import evaluate_predictions, write_evaluation_assets


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Phase4 retrieval mainline eval entrypoint")
    parser.add_argument("--manifest", required=True)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    manifest = load_manifest(args.manifest)
    config = load_config(manifest.config_path)
    predictions = load_predictions(manifest.predictions_path)
    metrics = evaluate_predictions(manifest, config, predictions)
    write_evaluation_assets(manifest, metrics)
    print(json.dumps({"run_id": manifest.run_id, "primary_metric": metrics.primary_metric, "values": metrics.values}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
