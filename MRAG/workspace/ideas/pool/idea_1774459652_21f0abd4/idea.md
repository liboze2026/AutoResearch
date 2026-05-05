# Query-Conditioned Layout Token Retrieval for Visual Documents

- Status: draft
- Source Type: auto
- Weight: 44
- Priority: 0
- Confidence: 0.44

Build a small retrieval baseline that converts each page into a compact sequence of layout tokens (text blocks, figures, tables, reading order, and coarse geometry) and matches queries against those tokens with a lightweight dual encoder. The scope is intentionally narrow: one encoder for the query, one encoder for page layout, and late interaction scoring, aimed at improving visual document retrieval when OCR text alone is weak.

## Structured Fields
- Research Direction: Use page-level visual/layout representations as the primary retrieval signal, with a controlled ablation against text-only and image-only baselines on a benchmark like VisDoM.
- Innovation Type: Representation + retrieval baseline
- Expected Advantage: Should outperform text-only retrieval on visually anchored queries by preserving spatial structure and non-text elements while remaining simple enough for a controlled experiment.
- Target Dataset Refs: VisDoM

## Risks
- The available paper parse is mock metadata, so benchmark details may need verification before implementation.
- If queries are mostly semantic text queries, gains from layout tokens may be small.
- Page-level tokenization may miss fine-grained figure or table content.

## Sources
- idea generator insight event source
- dataset_asset:VisDoM
- dataset_eval_protocol:Recall@1
- dataset_eval_protocol:Recall@5
- dataset_eval_protocol:MRR
- dataset_eval_protocol:nDCG@10
