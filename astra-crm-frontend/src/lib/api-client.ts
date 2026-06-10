import type { ApiSchema } from "@/lib/generated/openapi";

export type ApiErrorPayload = {
  error?: string | ApiSchema<"ErrorResponse">["error"];
  message?: string;
  requestId?: string;
};

export class ApiError extends Error {
  readonly status: number;
  readonly requestId?: string;

  constructor(message: string, status: number, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.requestId = requestId;
  }
}

export type ApiClientOptions = {
  baseUrl?: string;
};

export class ApiClient {
  private readonly baseUrl: string;

  constructor(options: ApiClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? import.meta.env.VITE_API_BASE_URL ?? "/api/v1";
  }

  async get<T>(path: string, init?: RequestInit) {
    return this.request<T>(path, { ...init, method: "GET" });
  }

  async post<T>(path: string, body?: unknown, init?: RequestInit) {
    return this.request<T>(path, this.withJsonBody("POST", body, init));
  }

  async patch<T>(path: string, body?: unknown, init?: RequestInit) {
    return this.request<T>(path, this.withJsonBody("PATCH", body, init));
  }

  async delete<T>(path: string, init?: RequestInit) {
    return this.request<T>(path, { ...init, method: "DELETE" });
  }

  async upload<T>(path: string, formData: FormData, init?: RequestInit) {
    return this.request<T>(path, {
      ...init,
      method: "POST",
      body: formData,
    });
  }

  private async request<T>(path: string, init: RequestInit) {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      credentials: "include",
      headers: {
        Accept: "application/json",
        ...init.headers,
      },
    });

    if (!response.ok) {
      throw await toApiError(response);
    }

    if (response.status === 204) {
      return undefined as T;
    }

    return (await response.json()) as T;
  }

  private withJsonBody(method: string, body?: unknown, init?: RequestInit): RequestInit {
    return {
      ...init,
      method,
      headers: {
        "Content-Type": "application/json",
        ...init?.headers,
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    };
  }
}

export function queryString(params: Record<string, string | number | undefined | null>) {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "" || value === "all") {
      continue;
    }
    searchParams.set(key, String(value));
  }

  const value = searchParams.toString();
  return value ? `?${value}` : "";
}

async function toApiError(response: Response) {
  let payload: ApiErrorPayload = {};

  try {
    payload = (await response.json()) as ApiErrorPayload;
  } catch {
    payload = {};
  }

  const errorMessage = typeof payload.error === "object" ? payload.error.message : payload.error;
  const fieldMessages = typeof payload.error === "object" ? Object.values(payload.error.fields ?? {}) : [];
  const details = typeof payload.error === "object" ? payload.error.details ?? [] : [];
  const message =
    payload.message ??
    errorMessage ??
    fieldMessages[0] ??
    details[0] ??
    (response.status >= 500 ? "Сервис временно недоступен. Попробуйте позже." : "Не удалось выполнить действие.");

  return new ApiError(message, response.status, payload.requestId);
}

export const apiClient = new ApiClient();
