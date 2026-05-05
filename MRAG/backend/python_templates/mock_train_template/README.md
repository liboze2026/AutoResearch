# mock_train_template

This template is the minimal stage2 training template.

It is intentionally mock-first:

- `train.py` simulates a training phase
- `evaluate.py` writes deterministic evaluation artifacts
- `runner.py` orchestrates the flow from an ExperimentSpec

Expected artifacts:

- `metrics.json`
- `result.md`
- `stdout.log`
- `stderr.log`

The template keeps extension points for later real training code.
