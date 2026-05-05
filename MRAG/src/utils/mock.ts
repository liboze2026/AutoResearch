export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

export function withDelay<T>(data: T, ms = 300): Promise<ApiEnvelope<T>> {
  return new Promise((resolve) => {
    setTimeout(() => resolve({ code: 0, message: "ok", data }), ms);
  });
}

export function withError(message: string, code = 500): Promise<never> {
  return Promise.reject({ code, message });
}
