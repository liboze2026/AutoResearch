import { http } from "@/api/http";
import type {
  AgentArtifact,
  Phase4DatasetProfile,
  Phase4DatasetProfileCreateRequest,
  Phase4ReaderContext,
  Phase4ReaderSource,
  Phase4Idea,
  Phase4IdeaScoreView,
  Phase4RunManifest,
  Phase4StructuredReportRecord,
  Phase4Workflow,
  Phase4WorkflowCreateRequest,
  Phase4WorkflowDetail,
  Phase4WorkflowRetryStageRequest,
  Phase4WorkflowSelectIdeaRequest
} from "@/types/domain";

export interface Phase4ReaderJobDetail {
  artifacts: AgentArtifact[];
  readerContext?: Phase4ReaderContext;
  readerSources: Phase4ReaderSource[];
  warnings: string[];
}

export interface Phase4IdeaJobDetail {
  artifacts: AgentArtifact[];
  ideas: Phase4Idea[];
  topRecommendations: Phase4IdeaScoreView[];
  warnings: string[];
}

export interface Phase4CodingJobDetail {
  artifacts: AgentArtifact[];
  runManifest?: Phase4RunManifest;
  warnings: string[];
}

export interface Phase4WriterJobDetail {
  artifacts: AgentArtifact[];
  report?: Phase4StructuredReportRecord;
  warnings: string[];
}

export async function getPhase4DatasetProfiles(params?: {
  taskType?: string;
  status?: string;
}): Promise<Phase4DatasetProfile[]> {
  return http<Phase4DatasetProfile[]>("/phase4/dataset-profiles", {
    apiVersion: "v2",
    query: params
  });
}

export async function createPhase4DatasetProfile(payload: Phase4DatasetProfileCreateRequest): Promise<Phase4DatasetProfile> {
  return http<Phase4DatasetProfile>("/phase4/dataset-profiles", {
    apiVersion: "v2",
    method: "POST",
    body: payload
  });
}

export async function getPhase4DatasetProfileById(id: string): Promise<Phase4DatasetProfile> {
  return http<Phase4DatasetProfile>(`/phase4/dataset-profiles/${id}`, {
    apiVersion: "v2"
  });
}

export async function updatePhase4DatasetProfile(id: string, payload: Partial<Phase4DatasetProfileCreateRequest>): Promise<Phase4DatasetProfile> {
  return http<Phase4DatasetProfile>(`/phase4/dataset-profiles/${id}`, {
    apiVersion: "v2",
    method: "PATCH",
    body: payload
  });
}

export async function getPhase4ReaderSources(params?: {
  datasetProfileId?: string;
}): Promise<Phase4ReaderSource[]> {
  return http<Phase4ReaderSource[]>("/phase4/reader-sources", {
    apiVersion: "v2",
    query: params
  });
}

export async function getPhase4ReaderContexts(params?: {
  datasetProfileId?: string;
}): Promise<Phase4ReaderContext[]> {
  return http<Phase4ReaderContext[]>("/phase4/reader-contexts", {
    apiVersion: "v2",
    query: params
  });
}

export async function getPhase4Ideas(params?: {
  datasetProfileId?: string;
  status?: string;
}): Promise<Phase4Idea[]> {
  return http<Phase4Idea[]>("/phase4/ideas", {
    apiVersion: "v2",
    query: params
  });
}

export async function getPhase4IdeaById(id: string): Promise<Phase4Idea> {
  return http<Phase4Idea>(`/phase4/ideas/${id}`, {
    apiVersion: "v2"
  });
}

export async function updatePhase4Idea(id: string, payload: Partial<Phase4Idea>): Promise<Phase4Idea> {
  return http<Phase4Idea>(`/phase4/ideas/${id}`, {
    apiVersion: "v2",
    method: "PATCH",
    body: payload
  });
}

export async function deletePhase4Idea(id: string): Promise<void> {
  await http(`/phase4/ideas/${id}`, {
    apiVersion: "v2",
    method: "DELETE"
  });
}

export async function selectPhase4Idea(id: string): Promise<Phase4Idea> {
  return http<Phase4Idea>(`/phase4/ideas/${id}/select`, {
    apiVersion: "v2",
    method: "POST",
    body: {}
  });
}

export async function archivePhase4Idea(id: string): Promise<Phase4Idea> {
  return http<Phase4Idea>(`/phase4/ideas/${id}/archive`, {
    apiVersion: "v2",
    method: "POST",
    body: {}
  });
}

export async function rejectPhase4Idea(id: string): Promise<Phase4Idea> {
  return http<Phase4Idea>(`/phase4/ideas/${id}/reject`, {
    apiVersion: "v2",
    method: "POST",
    body: {}
  });
}

export async function getPhase4IdeaScoreViews(params?: {
  datasetProfileId?: string;
  status?: string;
}): Promise<Phase4IdeaScoreView[]> {
  return http<Phase4IdeaScoreView[]>("/phase4/ideas/score-view", {
    apiVersion: "v2",
    query: params
  });
}

export async function getPhase4Runs(params?: {
  datasetProfileId?: string;
  ideaId?: string;
  status?: string;
}): Promise<Phase4RunManifest[]> {
  return http<Phase4RunManifest[]>("/phase4/runs", {
    apiVersion: "v2",
    query: params
  });
}

export async function getPhase4RunById(id: string): Promise<Phase4RunManifest> {
  return http<Phase4RunManifest>(`/phase4/runs/${id}`, {
    apiVersion: "v2"
  });
}

export async function updatePhase4RunStatus(id: string, payload: {
  status: string;
  retryCount?: number;
  failureFeedback?: Record<string, unknown>;
}): Promise<Phase4RunManifest> {
  return http<Phase4RunManifest>(`/phase4/runs/${id}/status`, {
    apiVersion: "v2",
    method: "POST",
    body: payload
  });
}

export async function getPhase4Reports(params?: {
  runManifestId?: string;
}): Promise<Phase4StructuredReportRecord[]> {
  return http<Phase4StructuredReportRecord[]>("/phase4/reports", {
    apiVersion: "v2",
    query: params
  });
}

export async function getPhase4ReportById(id: string): Promise<Phase4StructuredReportRecord> {
  return http<Phase4StructuredReportRecord>(`/phase4/reports/${id}`, {
    apiVersion: "v2"
  });
}

export async function getPhase4Workflows(params?: {
  datasetProfileId?: string;
  status?: string;
}): Promise<Phase4Workflow[]> {
  return http<Phase4Workflow[]>("/phase4/workflows", {
    apiVersion: "v2",
    query: params
  });
}

export async function createPhase4Workflow(payload: Phase4WorkflowCreateRequest): Promise<Phase4WorkflowDetail> {
  return http<Phase4WorkflowDetail>("/phase4/workflows", {
    apiVersion: "v2",
    method: "POST",
    body: payload
  });
}

export async function getPhase4WorkflowById(id: string): Promise<Phase4WorkflowDetail> {
  return http<Phase4WorkflowDetail>(`/phase4/workflows/${id}`, {
    apiVersion: "v2"
  });
}

export async function selectPhase4WorkflowIdea(id: string, payload: Phase4WorkflowSelectIdeaRequest): Promise<Phase4WorkflowDetail> {
  return http<Phase4WorkflowDetail>(`/phase4/workflows/${id}/select-idea`, {
    apiVersion: "v2",
    method: "POST",
    body: payload
  });
}

export async function selectPhase4WorkflowRevision(id: string, payload: Phase4WorkflowSelectIdeaRequest): Promise<Phase4WorkflowDetail> {
  return http<Phase4WorkflowDetail>(`/phase4/workflows/${id}/select-revision`, {
    apiVersion: "v2",
    method: "POST",
    body: payload
  });
}

export async function retryPhase4WorkflowStage(id: string, payload: Phase4WorkflowRetryStageRequest): Promise<Phase4WorkflowDetail> {
  return http<Phase4WorkflowDetail>(`/phase4/workflows/${id}/retry-stage`, {
    apiVersion: "v2",
    method: "POST",
    body: payload
  });
}

export async function archivePhase4Workflow(id: string): Promise<Phase4WorkflowDetail> {
  return http<Phase4WorkflowDetail>(`/phase4/workflows/${id}/archive`, {
    apiVersion: "v2",
    method: "POST",
    body: {}
  });
}

export async function getPhase4ReaderJob(id: string): Promise<Phase4ReaderJobDetail> {
  return http<Phase4ReaderJobDetail>(`/phase4/reader/jobs/${id}`, {
    apiVersion: "v2"
  });
}

export async function getPhase4IdeaJob(id: string): Promise<Phase4IdeaJobDetail> {
  return http<Phase4IdeaJobDetail>(`/phase4/ideas/jobs/${id}`, {
    apiVersion: "v2"
  });
}

export async function getPhase4CodingJob(id: string): Promise<Phase4CodingJobDetail> {
  return http<Phase4CodingJobDetail>(`/phase4/coding/jobs/${id}`, {
    apiVersion: "v2"
  });
}

export async function getPhase4WriterJob(id: string): Promise<Phase4WriterJobDetail> {
  return http<Phase4WriterJobDetail>(`/phase4/writer/jobs/${id}`, {
    apiVersion: "v2"
  });
}
