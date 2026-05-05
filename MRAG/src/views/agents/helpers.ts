import type { AgentInputRef } from "@/types/domain";

export function formatAgentInputSummary(inputRefs?: AgentInputRef[]): string {
  if (!inputRefs?.length) {
    return "-";
  }
  return inputRefs
    .slice(0, 3)
    .map((item) => `${item.ref_type}${item.ref_id ? `:${item.ref_id}` : item.ref_path ? `:${item.ref_path}` : ""}`)
    .join(" | ");
}

export function formatAgentOutputSummary(payload?: Record<string, unknown>): string {
  if (!payload || Object.keys(payload).length === 0) {
    return "-";
  }
  if (typeof payload.summary === "string" && payload.summary.trim()) {
    return payload.summary.trim();
  }
  if (Array.isArray(payload.candidate_papers)) {
    return `candidate_papers: ${payload.candidate_papers.length}`;
  }
  if (Array.isArray(payload.novelty_points)) {
    return `novelty_points: ${payload.novelty_points.length}`;
  }
  if (Array.isArray(payload.figure_plan)) {
    return `figure_plan: ${payload.figure_plan.length}`;
  }
  if (typeof payload.title === "string" && payload.title.trim()) {
    return payload.title.trim();
  }
  return truncateJson(payload);
}

export function truncateJson(value: unknown, max = 160): string {
  const text = JSON.stringify(value ?? {}, null, 2);
  if (text.length <= max) {
    return text;
  }
  return `${text.slice(0, max)}...`;
}

export function formatTagList(items?: string[]): string[] {
  return (items || []).filter((item) => Boolean(item && item.trim()));
}
