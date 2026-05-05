# VisDoM Page-Region Hard Negative Retriever

- Status: draft
- Source Type: auto
- Weight: 66
- Priority: 0
- Confidence: 0.66

Train a lightweight retrieval adapter that reranks candidate pages/regions using OCR text plus coarse visual layout features, with hard negatives mined from visually similar but semantically different pages. The scope is limited to improving top-k visual document retrieval for multimodal RAG without changing the generator.

## Structured Fields
- Research Direction: Use the VisDoM benchmark as the primary testbed and add a small reranking module over baseline text/vision embeddings. Compare plain embedding retrieval vs. layout-aware reranking with mined hard negatives.
- Innovation Type: low-scope retrieval reranking
- Expected Advantage: Should reduce confusions between visually similar pages and improve top-k retrieval quality with minimal model and engineering overhead.
- Target Dataset Refs: VisDoM

## Risks
- May overfit to VisDoM layout patterns
- Hard-negative mining may be noisy if OCR is weak
- Gains may be small if the baseline retriever is already strong

## Sources
- idea generator insight event source
- dataset_asset:VisDoM
- dataset_eval_protocol:Recall@1/5/10 for query-to-page retrieval
- dataset_eval_protocol:MRR for ranked retrieval quality
- dataset_eval_protocol:End-to-end answer accuracy with top-k retrieved pages as context
