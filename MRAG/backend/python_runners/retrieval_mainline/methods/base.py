from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any


@dataclass
class RetrievalMethod:
    name: str
    method_tags: list[str] = field(default_factory=list)
    score_bias: float = 0.0
    retrieval_notes: list[str] = field(default_factory=list)

    def run(self, manifest: Any, config: Any, dataset_asset: dict[str, Any]) -> dict[str, Any]:
        raise NotImplementedError
