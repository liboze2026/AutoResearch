import { http } from "@/api/http";
import type {
  Dataset,
  DatasetDetail,
  DatasetImportRequest,
  DatasetIndexTask,
  DatasetPathValidationRequest,
  DatasetPathValidationResult,
  DatasetUpdateRequest,
  ServerDatasetScanRequest,
  ServerDatasetScanResult
} from "@/types/domain";

export async function getDatasets(params?: {
  keyword?: string;
  sourceType?: "local" | "remote";
  modality?: string;
}): Promise<Dataset[]> {
  return http<Dataset[]>("/datasets", { query: params });
}

export async function getDatasetById(id: string): Promise<DatasetDetail | undefined> {
  return http<DatasetDetail>(`/datasets/${id}`);
}

export async function validateDatasetPath(payload: DatasetPathValidationRequest): Promise<DatasetPathValidationResult> {
  return http<DatasetPathValidationResult>("/datasets/validate-path", {
    method: "POST",
    body: payload
  });
}

export async function importDataset(payload: DatasetImportRequest): Promise<DatasetDetail> {
  return http<DatasetDetail>("/datasets", {
    method: "POST",
    body: payload
  });
}

export async function updateDataset(id: string, payload: DatasetUpdateRequest): Promise<DatasetDetail> {
  return http<DatasetDetail>(`/datasets/${id}`, {
    method: "PUT",
    body: payload
  });
}

export async function buildDatasetIndex(id: string): Promise<DatasetIndexTask> {
  return http<DatasetIndexTask>(`/datasets/${id}/build-index`, {
    method: "POST",
    body: {}
  });
}

export async function syncDatasetIndexTask(id: string, taskId: string): Promise<DatasetIndexTask> {
  return http<DatasetIndexTask>(`/datasets/${id}/index-tasks/${taskId}/sync`, {
    method: "POST",
    body: {}
  });
}

export async function scanServerDatasets(payload: ServerDatasetScanRequest): Promise<ServerDatasetScanResult> {
  return http<ServerDatasetScanResult>(`/servers/${payload.serverId}/scan-datasets`, {
    method: "POST",
    body: {
      rootPath: payload.rootPath,
      maxDepth: payload.maxDepth
    }
  });
}
