#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path

from common import read_text, write_json, write_text


def extract_title(parsed_text: str) -> str:
    for line in parsed_text.splitlines():
        candidate = line.strip().lstrip("\ufeff")
        if candidate.startswith("#"):
            return candidate.lstrip("# ").strip()
    return "Untitled Paper"


def deterministic_summary(title: str) -> str:
    return (
        f"{title} is summarized by the stage 1 mock insight extractor as a paper about "
        f"structured research asset management and controllable parsing workflows."
    )


def deterministic_contributions(title: str) -> list[str]:
    return [
        f"Frames {title} as a manageable research asset.",
        "Separates archival research objects from execution automation.",
    ]


def deterministic_methods() -> list[str]:
    return [
        "Use deterministic workspace artifacts for auditability.",
        "Use schema-first ingestion before deeper automation.",
    ]


def deterministic_limitations() -> list[str]:
    return [
        "Current parser and extractor are mock and deterministic.",
        "No real NLP or PDF structure recovery is performed in stage 1.",
    ]


def deterministic_novelty_points(title: str) -> list[str]:
    return [
        f"Highlights {title} as a controllable research asset pipeline example.",
        "Emphasizes schema-first automation before full autonomy.",
    ]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Deterministic mock insight extractor for stage 1")
    parser.add_argument("--parsed-paper", required=True)
    parser.add_argument("--output-dir", required=True)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    parsed_paper = Path(args.parsed_paper)
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    parsed_text = read_text(parsed_paper)
    title = extract_title(parsed_text)
    summary = deterministic_summary(title)
    contributions = deterministic_contributions(title)
    methods = deterministic_methods()
    limitations = deterministic_limitations()
    novelty_points = deterministic_novelty_points(title)

    summary_path = output_dir / "summary.md"
    contributions_path = output_dir / "contributions.json"
    methods_path = output_dir / "methods.json"
    limitations_path = output_dir / "limitations.json"
    novelty_points_path = output_dir / "novelty_points.json"

    write_text(summary_path, summary + "\n")
    write_json(contributions_path, contributions)
    write_json(methods_path, methods)
    write_json(limitations_path, limitations)
    write_json(novelty_points_path, novelty_points)

    print(
        json.dumps(
            {
                "status": "ok",
                "extract_mode": "python_mock_v1",
                "mock_extracted": True,
                "summary_path": str(summary_path),
                "contributions_path": str(contributions_path),
                "methods_path": str(methods_path),
                "limitations_path": str(limitations_path),
                "novelty_points_path": str(novelty_points_path),
                "summary": summary,
                "contributions": contributions,
                "methods": methods,
                "limitations": limitations,
                "novelty_points": novelty_points,
            },
            ensure_ascii=False,
        )
    )


if __name__ == "__main__":
    main()

