#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path

from common import read_text, write_json, write_text


def humanize_title(file_name: str) -> str:
    stem = Path(file_name).stem.replace("_", " ").replace("-", " ").strip().lstrip("\ufeff")
    if not stem:
        return "Untitled Paper"
    return " ".join(part[:1].upper() + part[1:].lower() for part in stem.split())


def maybe_extract_heading(text: str) -> str:
    for line in text.splitlines():
        candidate = line.strip().lstrip("\ufeff")
        if candidate.startswith("#"):
            return candidate.lstrip("# ").strip()
        if len(candidate) >= 12:
            return candidate
    return ""


def build_abstract(title: str) -> str:
    return (
        f"Mock abstract for {title}. This stage 1 parser produces deterministic "
        f"metadata and markdown artifacts and is intended to be replaced by a real PDF parser later."
    )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Deterministic mock paper parser for stage 1")
    parser.add_argument("--paper-file", required=True)
    parser.add_argument("--output-dir", required=True)
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    paper_file = Path(args.paper_file)
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    text_preview = ""
    try:
        text_preview = read_text(paper_file)
    except UnicodeDecodeError:
        text_preview = ""
    except FileNotFoundError as exc:
        raise SystemExit(str(exc))

    title = maybe_extract_heading(text_preview) or humanize_title(paper_file.name)
    abstract = build_abstract(title)
    authors = "Mock Author"
    venue = ""
    year = 0
    parser_note = "Current parsing is mock-based: title comes from heading or file name, abstract/authors are placeholders."

    metadata = {
        "title": title,
        "abstract": abstract,
        "authors": authors,
        "venue": venue,
        "year": year,
        "status": "parsed",
        "parse_mode": "python_mock_v1",
        "mock_parsed": True,
        "parser_note": parser_note,
        "paper_file": str(paper_file),
    }
    parsed_md = "\n".join(
        [
            f"# {title}",
            "",
            "## Abstract",
            abstract,
            "",
            "## Authors",
            authors,
            "",
            "## Source",
            str(paper_file),
            "",
            "## Parser Note",
            parser_note,
            "",
        ]
    )

    metadata_path = output_dir / "metadata.json"
    parsed_path = output_dir / "parsed.md"
    write_json(metadata_path, metadata)
    write_text(parsed_path, parsed_md)

    print(
        json.dumps(
            {
                "status": "ok",
                "title": title,
                "abstract": abstract,
                "authors": authors,
                "venue": venue,
                "year": year,
                "parse_mode": "python_mock_v1",
                "mock_parsed": True,
                "parser_note": parser_note,
                "metadata_path": str(metadata_path),
                "parsed_path": str(parsed_path),
            },
            ensure_ascii=False,
        )
    )


if __name__ == "__main__":
    main()

