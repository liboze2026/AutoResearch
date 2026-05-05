import { http } from "@/api/http";
import type {
  AgentArtifact,
  AgentEvent,
  AgentJob,
  AgentSummary,
  SkillDefinition,
  ToolDefinition
} from "@/types/domain";

export function getAgents(): Promise<AgentSummary[]> {
  return http<AgentSummary[]>("/agents");
}

export function getAgentJobs(limit = 100): Promise<AgentJob[]> {
  return http<AgentJob[]>("/agents/jobs", {
    query: { limit }
  });
}

export function getAgentJobById(id: string): Promise<AgentJob> {
  return http<AgentJob>(`/agents/jobs/${id}`);
}

export function getAgentArtifacts(jobId: string): Promise<AgentArtifact[]> {
  return http<AgentArtifact[]>(`/agents/artifacts/${jobId}`);
}

export function getAgentEvents(limit = 100): Promise<AgentEvent[]> {
  return http<AgentEvent[]>("/agent-events", {
    query: { limit }
  });
}

export function triggerAgentJob(id: string): Promise<AgentJob> {
  return http<AgentJob>(`/agent-jobs/${id}/trigger`, {
    method: "POST",
    body: {}
  });
}

export function getTools(): Promise<ToolDefinition[]> {
  return http<ToolDefinition[]>("/tools");
}

export function getSkills(): Promise<SkillDefinition[]> {
  return http<SkillDefinition[]>("/skills");
}
