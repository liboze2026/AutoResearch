export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

const API_BASE = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080/api/v1";

function resolveApiBase(apiVersion?: "v1" | "v2"): string {
  if (apiVersion === "v2") {
    if (API_BASE.endsWith("/api/v1")) {
      return API_BASE.replace(/\/api\/v1$/, "/api/v2");
    }
    return API_BASE.replace(/\/api(?:\/v1)?$/, "/api/v2");
  }
  return API_BASE;
}

function buildQuery(query?: Record<string, string | number | boolean | undefined>): string {
  if (!query) {
    return "";
  }
  const params = new URLSearchParams();
  Object.entries(query).forEach(([k, v]) => {
    if (v === undefined || v === null || v === "") {
      return;
    }
    params.append(k, String(v));
  });
  const s = params.toString();
  return s ? `?${s}` : "";
}

export async function http<T>(path: string, options?: {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  query?: Record<string, string | number | boolean | undefined>;
  body?: unknown;
  apiVersion?: "v1" | "v2";
}): Promise<T> {
  const url = `${resolveApiBase(options?.apiVersion)}${path}${buildQuery(options?.query)}`;
  const res = await fetch(url, {
    method: options?.method || "GET",
    headers: {
      "Content-Type": "application/json"
    },
    body: options?.body ? JSON.stringify(options.body) : undefined
  });

  const envelope = (await res.json()) as ApiEnvelope<T>;
  if (!res.ok || envelope.code !== 0) {
    throw new Error(envelope.message || `HTTP ${res.status}`);
  }
  return envelope.data;
}

export async function httpForm<T>(path: string, formData: FormData, method: "POST" | "PATCH" = "POST", apiVersion?: "v1" | "v2"): Promise<T> {
  const url = `${resolveApiBase(apiVersion)}${path}`;
  const res = await fetch(url, {
    method,
    body: formData
  });
  const envelope = (await res.json()) as ApiEnvelope<T>;
  if (!res.ok || envelope.code !== 0) {
    throw new Error(envelope.message || `HTTP ${res.status}`);
  }
  return envelope.data;
}
