import argparse
import json
from pathlib import Path


def build_metrics(spec: dict) -> dict:
    expected = spec.get("expected_metrics", {})
    primary = expected.get("primary", "accuracy")
    return {
        "primary_metric": primary,
        "values": {
            primary: 0.88,
            "loss": 0.12,
            "latency_ms": 37
        },
        "comparison_targets_count": len(spec.get("comparison_targets", [])),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--spec", required=True)
    parser.add_argument("--output-dir", required=True)
    args = parser.parse_args()

    spec_path = Path(args.spec)
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    spec = json.loads(spec_path.read_text(encoding="utf-8"))
    metrics = build_metrics(spec)

    print("[mock-eval] evaluating outputs")

    (output_dir / "metrics.json").write_text(
        json.dumps(metrics, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    result_md = "\n".join([
        "# Mock Experiment Result",
        "",
        f"- Model: {spec.get('model_name', 'mock/model')}",
        f"- Primary Metric: {metrics['primary_metric']}",
        f"- Value: {metrics['values'][metrics['primary_metric']]}",
        f"- Comparison Targets: {metrics['comparison_targets_count']}",
        "",
        "This result is produced by the stage2 mock training template."
    ])
    (output_dir / "result.md").write_text(result_md + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
