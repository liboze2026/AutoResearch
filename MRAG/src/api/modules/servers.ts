import { http } from "@/api/http";
import type {
  GPUProbeResult,
  ServerNode,
  ServerNodePayload,
  ServerStatusSnapshot,
  SSHConnectionTestResult
} from "@/types/domain";

export async function getServers(): Promise<ServerNode[]> {
  return http<ServerNode[]>("/servers");
}

export async function createServer(payload: ServerNodePayload): Promise<ServerNode> {
  return http<ServerNode>("/servers", {
    method: "POST",
    body: payload
  });
}

export async function updateServer(id: string, payload: ServerNodePayload): Promise<ServerNode> {
  return http<ServerNode>(`/servers/${id}`, {
    method: "PUT",
    body: payload
  });
}

export async function deleteServer(id: string): Promise<{ deleted: boolean }> {
  return http<{ deleted: boolean }>(`/servers/${id}`, {
    method: "DELETE"
  });
}

export async function testServerConnection(id: string): Promise<SSHConnectionTestResult> {
  return http<SSHConnectionTestResult>(`/servers/${id}/test-connection`, {
    method: "POST",
    body: {}
  });
}

export async function checkServerGpu(id: string): Promise<GPUProbeResult> {
  return http<GPUProbeResult>(`/servers/${id}/check-gpu`, {
    method: "POST",
    body: {}
  });
}

export async function refreshServerStatus(id: string): Promise<ServerStatusSnapshot> {
  return http<ServerStatusSnapshot>(`/servers/${id}/refresh-status`, {
    method: "POST",
    body: {}
  });
}
