import argparse
import json
import shutil
import subprocess
import sys
from pathlib import Path


def validate_spec(spec: dict, schema: dict) -> None:
    missing = [field for field in schema.get("required_fields", []) if field not in spec]
    if missing:
        raise ValueError(f"missing required spec fields: {', '.join(missing)}")


def run_step(script: Path, spec_path: Path, output_dir: Path, stdout_log: Path, stderr_log: Path) -> None:
    cmd = [sys.executable, str(script), "--spec", str(spec_path), "--output-dir", str(output_dir)]
    completed = subprocess.run(cmd, capture_output=True, text=True, check=False)
    with stdout_log.open("a", encoding="utf-8") as stdout_fp:
        if completed.stdout:
            stdout_fp.write(completed.stdout)
    with stderr_log.open("a", encoding="utf-8") as stderr_fp:
        if completed.stderr:
            stderr_fp.write(completed.stderr)
    if completed.returncode != 0:
        raise RuntimeError(f"{script.name} failed with exit code {completed.returncode}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--spec", required=True)
    parser.add_argument("--output-dir", required=True)
    args = parser.parse_args()

    spec_path = Path(args.spec)
    output_dir = Path(args.output_dir)
    template_dir = Path(__file__).resolve().parent
    output_dir.mkdir(parents=True, exist_ok=True)

    schema = json.loads((template_dir / "config_schema.json").read_text(encoding="utf-8"))
    spec = json.loads(spec_path.read_text(encoding="utf-8"))
    validate_spec(spec, schema)

    stdout_log = output_dir / "stdout.log"
    stderr_log = output_dir / "stderr.log"
    stdout_log.write_text("", encoding="utf-8")
    stderr_log.write_text("", encoding="utf-8")

    run_step(template_dir / "train.py", spec_path, output_dir, stdout_log, stderr_log)
    run_step(template_dir / "evaluate.py", spec_path, output_dir, stdout_log, stderr_log)

    shutil.copyfile(spec_path, output_dir / "used_spec.json")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
