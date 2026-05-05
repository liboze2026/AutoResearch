import { http } from "@/api/http";
import type {
  Experiment,
  ExperimentCreateRequest,
  ExperimentDetail,
  ExperimentQueueResult,
  ExperimentRun,
  ExperimentSpecDetail,
  GPUResourceSnapshot,
  ResultComparison,
  RunCompareResult,
  RunLog,
  RunRecoveryDetail,
  ScheduleResult,
  SchedulerDecision,
  ServerHeartbeat,
  SSHConnectionTestResult
} from "@/types/domain";

export function getExperiments(): Promise<Experiment[]> {
  return http<Experiment[]>("/experiments");
}

export function createExperiment(payload: ExperimentCreateRequest): Promise<ExperimentDetail> {
  return http<ExperimentDetail>("/experiments", {
    method: "POST",
    body: payload
  });
}

export function getExperimentById(id: string): Promise<ExperimentDetail> {
  return http<ExperimentDetail>(`/experiments/${id}`);
}

export function generateExperimentSpec(id: string): Promise<ExperimentSpecDetail> {
  return http<ExperimentSpecDetail>(`/experiments/${id}/generate-spec`, {
    method: "POST",
    body: {}
  });
}

export function getExperimentSpec(id: string): Promise<ExperimentSpecDetail> {
  return http<ExperimentSpecDetail>(`/experiments/${id}/spec`);
}

export function queueExperiment(id: string): Promise<ExperimentQueueResult> {
  return http<ExperimentQueueResult>(`/experiments/${id}/queue`, {
    method: "POST",
    body: {}
  });
}

export function getExperimentComparisons(id: string): Promise<ResultComparison[]> {
  return http<ResultComparison[]>(`/experiments/${id}/comparisons`);
}

export function getRunById(id: string): Promise<ExperimentRun> {
  return http<ExperimentRun>(`/runs/${id}`);
}

export function scheduleRun(id: string): Promise<ScheduleResult> {
  return http<ScheduleResult>(`/runs/${id}/schedule`, {
    method: "POST",
    body: {}
  });
}

export function startRun(id: string): Promise<ExperimentRun> {
  return http<ExperimentRun>(`/runs/${id}/start`, {
    method: "POST",
    body: {}
  });
}

export function retryRun(id: string): Promise<ExperimentQueueResult> {
  return http<ExperimentQueueResult>(`/runs/${id}/retry`, {
    method: "POST",
    body: {}
  });
}

export function compareRun(id: string): Promise<RunCompareResult> {
  return http<RunCompareResult>(`/runs/${id}/compare`, {
    method: "POST",
    body: {}
  });
}

export function getRunLogs(id: string): Promise<RunLog[]> {
  return http<RunLog[]>(`/runs/${id}/logs`);
}

export function getRunLogTail(id: string, logType = "stdout"): Promise<{ tail: string }> {
  return http<{ tail: string }>(`/runs/${id}/logs/tail`, {
    query: { type: logType }
  });
}

export function getRunRecovery(id: string): Promise<RunRecoveryDetail> {
  return http<RunRecoveryDetail>(`/runs/${id}/recovery`);
}

export function getSchedulerDecision(id: string): Promise<SchedulerDecision> {
  return http<SchedulerDecision>(`/runs/${id}/scheduler-decision`);
}

export function triggerServerHeartbeat(id: string): Promise<{ heartbeat: ServerHeartbeat; probe?: SSHConnectionTestResult }> {
  return http<{ heartbeat: ServerHeartbeat; probe?: SSHConnectionTestResult }>(`/servers/${id}/heartbeat`, {
    method: "POST",
    body: {}
  });
}

export function getServerHeartbeats(id: string): Promise<ServerHeartbeat[]> {
  return http<ServerHeartbeat[]>(`/servers/${id}/heartbeats`);
}

export function triggerServerGpuSnapshot(id: string): Promise<{
  serverId: string;
  capturedAt: string;
  availableGpuCount: number;
  totalGpuCount: number;
  snapshots: GPUResourceSnapshot[];
}> {
  return http<{
    serverId: string;
    capturedAt: string;
    availableGpuCount: number;
    totalGpuCount: number;
    snapshots: GPUResourceSnapshot[];
  }>(`/servers/${id}/gpu-snapshot`, {
    method: "POST",
    body: {}
  });
}

export function getServerGpuSnapshots(id: string): Promise<GPUResourceSnapshot[]> {
  return http<GPUResourceSnapshot[]>(`/servers/${id}/gpu-snapshots`);
}
