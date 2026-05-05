#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json


def main() -> None:
    parser = argparse.ArgumentParser(description="Stage 1 idea generator placeholder")
    parser.add_argument("--workspace", required=True)
    parser.add_argument("--source-id", required=False, default="")
    args = parser.parse_args()

    print(json.dumps({
        "status": "not_implemented",
        "workspace": args.workspace,
        "source_id": args.source_id,
        "message": "Stage 1 structure scaffold only."
    }, ensure_ascii=False))


if __name__ == "__main__":
    main()
