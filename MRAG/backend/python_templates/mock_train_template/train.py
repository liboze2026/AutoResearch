import argparse
import json
import time
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--spec", required=True)
    parser.add_argument("--output-dir", required=True)
    args = parser.parse_args()

    spec_path = Path(args.spec)
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)

    spec = json.loads(spec_path.read_text(encoding="utf-8"))
    train_state = {
        "stage": "train",
        "model_name": spec.get("model_name", "mock/model"),
        "template_type": spec.get("train_template_type", "mock_train_template"),
        "dataset_asset_id": spec.get("dataset_ref", {}).get("dataset_asset_id", ""),
        "hyperparams": spec.get("hyperparams", {}),
        "status": "completed"
    }

    print(f"[mock-train] start model={train_state['model_name']}")
    time.sleep(0.05)
    print("[mock-train] completed")

    (output_dir / "train_state.json").write_text(
        json.dumps(train_state, indent=2, ensure_ascii=False),
        encoding="utf-8",
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
