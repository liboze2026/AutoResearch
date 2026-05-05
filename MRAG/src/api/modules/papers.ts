import { http, httpForm } from "@/api/http";
import type {
  Paper,
  PaperDetail,
  PaperFile,
  PaperImportResult,
  PaperInsight,
  PaperInsightExtractionResult,
  PaperParseResult
} from "@/types/domain";

export function getPapers(): Promise<Paper[]> {
  return http<Paper[]>("/papers");
}

export function getPaperById(id: string): Promise<PaperDetail> {
  return http<PaperDetail>(`/papers/${id}`);
}

export function getPaperFiles(id: string): Promise<PaperFile[]> {
  return http<PaperFile[]>(`/papers/${id}/files`);
}

export function getPaperInsights(id: string): Promise<PaperInsight[]> {
  return http<PaperInsight[]>(`/papers/${id}/insights`);
}

export function importPaperFromWorkspace(existingPath: string): Promise<PaperImportResult> {
  return http<PaperImportResult>("/papers/import", {
    method: "POST",
    body: { existingPath }
  });
}

export function importPaperFromFile(file: File): Promise<PaperImportResult> {
  const formData = new FormData();
  formData.append("file", file);
  return httpForm<PaperImportResult>("/papers/import", formData, "POST");
}

export function parsePaper(id: string): Promise<PaperParseResult> {
  return http<PaperParseResult>(`/papers/${id}/parse`, {
    method: "POST",
    body: {}
  });
}

export function extractPaperInsights(id: string): Promise<PaperInsightExtractionResult> {
  return http<PaperInsightExtractionResult>(`/papers/${id}/extract-insights`, {
    method: "POST",
    body: {}
  });
}