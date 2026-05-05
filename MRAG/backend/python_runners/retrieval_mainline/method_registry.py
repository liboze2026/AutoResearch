from __future__ import annotations

import importlib.util
from pathlib import Path

from methods.dummy_retrieval import DummyRetrievalMethod


def load_method(module_path: str):
    module_path = (module_path or "").strip()
    if not module_path:
        return DummyRetrievalMethod(name="dummy_retrieval", method_tags=["default"], score_bias=0.0)
    path = Path(module_path)
    if not path.exists():
        return DummyRetrievalMethod(name="dummy_retrieval", method_tags=["fallback"], score_bias=0.0)
    spec = importlib.util.spec_from_file_location(path.stem, path)
    if spec is None or spec.loader is None:
        return DummyRetrievalMethod(name="dummy_retrieval", method_tags=["fallback"], score_bias=0.0)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    if hasattr(module, "build_method"):
        return module.build_method()
    return DummyRetrievalMethod(name="dummy_retrieval", method_tags=["fallback"], score_bias=0.0)
