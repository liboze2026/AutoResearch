import type {
  Dataset,
  Phase4DatasetProfile,
  Phase4Idea,
  Phase4ReaderContext,
  Phase4ReaderSource,
  Phase4StructuredReportRecord
} from "@/types/domain";

export function toPrettyJson(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}

export function parseJsonText<T>(value: string, fallback: T): T {
  const text = value.trim();
  if (!text) {
    return fallback;
  }
  return JSON.parse(text) as T;
}

export function parseLineList(value: string) {
  return value
    .split(/\r?\n|,/)
    .map((item) => item.trim())
    .filter(Boolean);
}

export function joinLineList(items: string[] | undefined) {
  return (items || []).join("\n");
}

export function joinCommaList(items: string[] | undefined) {
  return (items || []).join(", ");
}

export function downloadTextFile(filename: string, content: string, contentType = "text/plain;charset=utf-8") {
  const blob = new Blob([content], { type: contentType });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function matchPhase4DatasetProfile(profiles: Phase4DatasetProfile[], dataset?: Dataset | null) {
  if (!dataset) {
    return undefined;
  }
  return profiles.find((item) => {
    const sourceDatasetId = String(item.metadata?.sourceDatasetId || "");
    if (sourceDatasetId && sourceDatasetId === dataset.id) {
      return true;
    }
    return item.serverId === dataset.serverId && item.serverPath === dataset.path;
  });
}

export function overallIdeaScore(idea?: Pick<Phase4Idea, "scoreSummary"> | null) {
  const value = Number(idea?.scoreSummary?.overallScore || 0);
  return value ? value : 0;
}

export function sortPhase4Ideas(items: Phase4Idea[], sortBy: string, scoreMap: Record<string, number>) {
  const cloned = [...items];
  cloned.sort((left, right) => {
    if (sortBy === "title") {
      return left.title.localeCompare(right.title);
    }
    if (sortBy === "updatedAt") {
      return new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime();
    }
    if (sortBy === "status") {
      return left.status.localeCompare(right.status);
    }
    return (scoreMap[right.id] || overallIdeaScore(right)) - (scoreMap[left.id] || overallIdeaScore(left));
  });
  return cloned;
}

export function readerCitationRows(context?: Phase4ReaderContext | null, sources: Phase4ReaderSource[] = []) {
  const sourceMap = new Map(sources.map((item) => [item.id, item]));
  const orderedSources = (context?.sourceIds || [])
    .map((id) => sourceMap.get(id))
    .filter((item): item is Phase4ReaderSource => !!item);
  const structured = Array.isArray(context?.structuredContext?.citation_metadata)
    ? (context?.structuredContext?.citation_metadata as Record<string, unknown>[])
    : [];
  return {
    orderedSources,
    citationMetadata: structured
  };
}

export function extractReportMetrics(report?: Phase4StructuredReportRecord | null) {
  const values = (report?.machineReadableReport?.metrics as Record<string, unknown> | undefined)?.values;
  if (!values || typeof values !== "object") {
    return [];
  }
  return Object.entries(values as Record<string, unknown>).map(([key, value]) => ({
    key,
    value
  }));
}

export function extractReportErrorSummary(report?: Phase4StructuredReportRecord | null) {
  const summary = report?.machineReadableReport?.error_summary;
  return summary && typeof summary === "object" ? (summary as Record<string, unknown>) : {};
}

export function extractReportArtifactSummary(report?: Phase4StructuredReportRecord | null) {
  const implementation = report?.machineReadableReport?.implementation;
  const artifactSummary = implementation && typeof implementation === "object"
    ? (implementation as Record<string, unknown>).artifact_summary
    : undefined;
  return artifactSummary && typeof artifactSummary === "object" ? (artifactSummary as Record<string, unknown>) : {};
}
