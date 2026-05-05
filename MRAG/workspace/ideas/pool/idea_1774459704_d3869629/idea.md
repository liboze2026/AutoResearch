# Late-Fusion OCR + Vision Retriever for VisDoM

- Status: draft
- Source Type: auto
- Weight: 77
- Priority: 0
- Confidence: 0.77

A minimal multimodal RAG baseline for VisDoM that indexes each document page with two signals: a lightweight visual embedding from the page image and an OCR-text embedding from extracted text, then retrieves by late-fusing the scores. The experiment is intentionally small and stage3-friendly: compare image-only, OCR-only, and fused retrieval on the existing VisDoM split to test whether adding cheap text signals improves document retrieval without a heavy cross-encoder.

## Structured Fields
- Research Direction: Build a baseline-friendly multimodal retrieval pipeline for visual document search using dual encoders and late fusion, then validate whether OCR complements page appearance on VisDoM.
- Innovation Type: simple_multimodal_baseline
- Expected Advantage: Low compute, easy to implement on a single GPU, and likely stronger than pure visual retrieval when queries depend on document text cues.
- Target Dataset Refs: dasset_1774459658_85800333

## Risks
- OCR quality may dominate results and obscure the visual contribution.
- Late fusion may yield only modest gains if VisDoM queries are mostly appearance-based.
- Single-stage retrieval may underperform more complex rerankers on harder cases.

## Sources
- idea generator paper insight source (VisDoM: A Benchmark for Visual Document Retrieval)
- dataset_asset:dasset_1774459658_85800333
- dataset_eval_protocol:D:\3\MRAG\workspace\datasets\dasset_1774459658_85800333\evalplan.json
- human_hint: Focus on a minimal controlled multimodal RAG experiment grounded on the VisDoM benchmark.
- human_hint: Prefer a small baseline-friendly idea that can run on a single idle GPU on shenzhenvlab.
- human_hint: Keep the experiment lightweight and suitable for stage3 validation rather than SOTA performance.
