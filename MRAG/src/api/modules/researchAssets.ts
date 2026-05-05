import { http } from "@/api/http";
import type {
  Baseline,
  BaselineCreateRequest,
  BaselineDetail,
  BaselineUpdateRequest,
  DatasetAsset,
  DatasetAssetCreateRequest,
  DatasetAssetDetail,
  DatasetAssetRegisterFromScanRequest,
  ResultArchive,
  ResultArchiveCreateRequest,
  ResultArchiveDetail,
  ResultArchiveUpdateRequest
} from "@/types/domain";

export function getDatasetAssets(): Promise<DatasetAsset[]> {
  return http<DatasetAsset[]>("/dataset-assets");
}

export function getDatasetAssetById(id: string): Promise<DatasetAssetDetail> {
  return http<DatasetAssetDetail>(`/dataset-assets/${id}`);
}

export function createDatasetAsset(payload: DatasetAssetCreateRequest): Promise<DatasetAssetDetail> {
  return http<DatasetAssetDetail>("/dataset-assets", {
    method: "POST",
    body: payload
  });
}

export function registerDatasetAssetFromScan(payload: DatasetAssetRegisterFromScanRequest): Promise<DatasetAssetDetail> {
  return http<DatasetAssetDetail>("/dataset-assets/register-from-scan", {
    method: "POST",
    body: payload
  });
}

export function getBaselines(): Promise<Baseline[]> {
  return http<Baseline[]>("/baselines");
}

export function getBaselineById(id: string): Promise<BaselineDetail> {
  return http<BaselineDetail>(`/baselines/${id}`);
}

export function createBaseline(payload: BaselineCreateRequest): Promise<BaselineDetail> {
  return http<BaselineDetail>("/baselines", {
    method: "POST",
    body: payload
  });
}

export function updateBaseline(id: string, payload: BaselineUpdateRequest): Promise<BaselineDetail> {
  return http<BaselineDetail>(`/baselines/${id}`, {
    method: "PATCH",
    body: payload
  });
}

export function getResultArchives(): Promise<ResultArchive[]> {
  return http<ResultArchive[]>("/result-archives");
}

export function getResultArchiveById(id: string): Promise<ResultArchiveDetail> {
  return http<ResultArchiveDetail>(`/result-archives/${id}`);
}

export function createResultArchive(payload: ResultArchiveCreateRequest): Promise<ResultArchiveDetail> {
  return http<ResultArchiveDetail>("/result-archives", {
    method: "POST",
    body: payload
  });
}

export function updateResultArchive(id: string, payload: ResultArchiveUpdateRequest): Promise<ResultArchiveDetail> {
  return http<ResultArchiveDetail>(`/result-archives/${id}`, {
    method: "PATCH",
    body: payload
  });
}