// Typed fetch wrapper. Adds Bearer header, parses {error, code} bodies,
// and transparently refreshes the access token once on 401.

import {
  clearAuth,
  getAccessToken,
  getRefreshToken,
  setTokens,
} from "./auth";
import type { ApiError, RefreshResponse } from "./types";

// API base. When NEXT_PUBLIC_API_URL is set to "" (production same-origin), all requests are
// made relative to the current host (e.g. https://invest.duoclieuhk.vn/api/v1/...), which nginx
// proxies to the backend. This removes any dependency on a separate api.* hostname / CORS.
// When the env var is undefined (local dev), fall back to the dev API on localhost.
export const API_URL =
  process.env.NEXT_PUBLIC_API_URL === undefined
    ? "http://localhost:8080"
    : process.env.NEXT_PUBLIC_API_URL;

export class ApiException extends Error {
  code?: string;
  status: number;
  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "ApiException";
    this.status = status;
    this.code = code;
  }
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  auth?: boolean;
  headers?: Record<string, string>;
}

async function parseError(res: Response): Promise<ApiError> {
  try {
    const data = (await res.json()) as ApiError;
    if (data && typeof data.error === "string") return data;
    return { error: `Request failed (${res.status})` };
  } catch {
    return { error: `Request failed (${res.status})` };
  }
}

let refreshing: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  if (refreshing) return refreshing;
  refreshing = (async () => {
    const refresh_token = getRefreshToken();
    if (!refresh_token) return false;
    try {
      const res = await fetch(`${API_URL}/api/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token }),
      });
      if (!res.ok) {
        clearAuth();
        return false;
      }
      const data = (await res.json()) as RefreshResponse;
      setTokens(data.tokens);
      return true;
    } catch {
      return false;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

export async function apiFetch<T>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const { method = "GET", body, auth = false, headers = {} } = options;

  const buildHeaders = (): Record<string, string> => {
    const h: Record<string, string> = { ...headers };
    if (body !== undefined && !(body instanceof FormData)) {
      h["Content-Type"] = "application/json";
    }
    if (auth) {
      const token = getAccessToken();
      if (token) h["Authorization"] = `Bearer ${token}`;
    }
    return h;
  };

  const doFetch = () =>
    fetch(`${API_URL}${path}`, {
      method,
      headers: buildHeaders(),
      body:
        body === undefined
          ? undefined
          : body instanceof FormData
          ? body
          : JSON.stringify(body),
    });

  let res = await doFetch();

  if (res.status === 401 && auth) {
    const ok = await tryRefresh();
    if (ok) {
      res = await doFetch();
    }
  }

  if (!res.ok) {
    const err = await parseError(res);
    throw new ApiException(err.error, res.status, err.code);
  }

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

// Convenience helper for a random idempotency key.
export function newIdempotencyKey(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `idemp-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}
