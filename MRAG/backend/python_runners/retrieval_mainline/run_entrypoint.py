#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json

from mainline import run_mainline


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Phase4 retrieval mainline run entrypoint")
    parser.add_argument("--manifest", required=True)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    result = run_mainline(args.manifest)
    print(json.dumps(result["summary"], ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
